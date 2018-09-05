package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/parth/go-gameloop"
)

type GameServer struct {
	Users           map[*Client]*Player
	Colors          map[*Snake]uint32
	World           *Map
	GameServerRedis *redis.Client
	PlayerRedis     *redis.Client
	ID              string
	GL              *gameLoop.GameLoop
	PlayerCount     int
	Pot             int
	SpectatorView   [][]uint32
	Leaderboard     []LeaderboardMessage
	Spectators      []*Client
	Minimap         []MinimapMessage
	LeaderboardSize int
}

var gameserver *GameServer

func main() {
	gameServerRedis := connectToRedis("redis-gameservers:6379")
	playerRedis := connectToRedis("redis-players:6379")
	id := os.Getenv("GSPORT")

	gameServerRedis.HSet(id, "status", "idle")

	players := getPlayers(id, gameServerRedis)
	pot := getPot(id, gameServerRedis)

	gameserver = &GameServer{
		Users:           make(map[*Client]*Player),
		Colors:          make(map[*Snake]uint32),
		World:           NewMap(players, 30, 20),
		GameServerRedis: gameServerRedis,
		PlayerRedis:     playerRedis,
		ID:              id,
		PlayerCount:     players,
		Pot:             pot,
	}

	gameserver.World.GameServer = gameserver

	if gameserver.PlayerCount < 10 {
		gameserver.LeaderboardSize = gameserver.PlayerCount
	} else {
		gameserver.LeaderboardSize = 10
	}

	gameserver.SpectatorView = make([][]uint32, len(gameserver.World.Tiles))

	for i := range gameserver.World.Tiles {
		gameserver.SpectatorView[i] = make([]uint32, len(gameserver.World.Tiles[i]))
	}

	gameserver.GL = gameLoop.New(5, gameserver.MapUpdater)

	http.HandleFunc("/ws", wsHandler)
	panic(http.ListenAndServe(":10000", nil))

}

func getPlayers(id string, gameServerRedis *redis.Client) int {
	for {
		playerCountString, _ := gameServerRedis.HGet(id, "players").Result()
		players, _ := strconv.Atoi(playerCountString)
		if players == 0 {
			time.Sleep(1000 * time.Millisecond)
		} else {
			gameServerRedis.HSet(id, "status", "ready")
			return players
		}
	}

	return 0
}

func getPot(id string, gameServerRedis *redis.Client) int {
	potString, err := gameServerRedis.HGet(id, "pot").Result()
	if err != nil {
		return 0
	}
	pot, _ := strconv.Atoi(potString)
	return pot
}

func connectToRedis(addr string) *redis.Client {
	var client *redis.Client
	for {
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
		})
		_, err := client.Ping().Result()
		if err != nil {
			fmt.Println("gameserver could not connect to redis")
			fmt.Println(err)
		} else {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("connected to redis")

	return client
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
	}

	gameserver.PlayerJoined(conn)
}

func (gs *GameServer) PlayerJoined(conn *websocket.Conn) {
	fmt.Println("player joined")
	message := &RegisterMessage{}

	error := conn.ReadJSON(message)

	if error == nil && validateToken(message.Token, gs.PlayerRedis) {
		c := NewClient(message, conn)
		c.Status = "in game"
		c.Player = &Player{}
		gs.World.SpawnNewPlayer(c.Player)
		c.Player.Client = c
		c.SendPot(gs)

		gs.Users[c] = c.Player
		go c.CollectInput(conn)

		fmt.Println(len(gs.Users), gs.PlayerCount)
		if len(gs.Users) >= gs.PlayerCount && gs.GL.Running == false {
			gs.GL.Start()
			fmt.Println("started")
		}
	} else {
		gs.Spectators = append(gs.Spectators, NewClient(nil, conn))
	}
}

// TODO OOP?
func validateToken(token string, playerRedis *redis.Client) bool {
	status, _ := playerRedis.HGet(token, "status").Result()
	if status == "paid" {
		playerRedis.HSet(token, "status", "in game")
		return true
	}
	return false
}

func (gs *GameServer) PublishState(msg string) {
	gs.GameServerRedis.HSet(gs.ID, "status", msg)
}

func (gs *GameServer) MapUpdater(delta float64) {
	gs.PublishState("game started")
	gs.World.Update()
	gs.CalculateLeaderboard()
	gs.CalculateSpectatorView()
	gs.CalculateMinimap()

	for player := range gs.World.Players {
		player.Client.SendPerspective(gs)
		player.Client.SendCustomLeaderboard(gs)
		player.Client.SendCustomMinimap(gs)
	}

	for loser := range gs.World.Losers {
		loser.Client.SendSpectatorView(gs)
		loser.Client.SendLeaderboard(gs)
		loser.Client.SendMinimap(gs)
	}

	for _, spectator := range gs.Spectators {
		spectator.SendSpectatorView(gs)
		spectator.SendLeaderboard(gs)
		spectator.SendMinimap(gs)
	}

	if len(gs.World.Players) == 1 {
		gs.PostGame()
		gs.PublishState("game finished")
		for player, _ := range gs.World.Players {
			gs.ClientWon(player.Client)
		}
		os.Exit(0)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func (gs *GameServer) ClientWon(client *Client) {
	client.Status = "won"
	gs.PlayerRedis.HSet(client.Token, "status", "won")
	client.SendStatus(gs)
}

func (gs *GameServer) ClientLost(client *Client) {
	client.Status = "lost"
	gs.PlayerRedis.HSet(client.Token, "status", "lost")
	gs.GameServerRedis.HSet(gs.ID, "players", len(gs.World.Players))
}

func (gs *GameServer) CalculateLeaderboard() {
	gs.LeaderboardSize = min(min(gs.PlayerCount, len(gs.World.Players)), 10)
	leaderboard := make([]LeaderboardMessage, len(gs.World.Players))
	snakes := gs.World.SortSnakes()
	for i, snake := range snakes {
		leaderboard[i] = NewLeaderboardMessage(i, gs, snake)
	}

	gs.Leaderboard = leaderboard
}

// What does using a tile as a fundemental game object look like.
// Maybe something like a Snakenode and a Food *are* tiles. And they
// Dictate what they look like.
func (gs *GameServer) GetColor(tile *Tile) uint32 {
	if tile.Food != nil {
		return 0x00FF00
	}

	if tile.Snake == nil {
		return 0xF0F0F0
	}

	if val, ok := gs.Colors[tile.Snake]; ok {
		return val
	}

	gs.Colors[tile.Snake] = rand.Uint32()
	return gs.Colors[tile.Snake]
}

func (gs *GameServer) CalculateSpectatorView() {
	for r := range gs.World.Tiles {
		for c := range gs.World.Tiles[r] {
			gs.SpectatorView[r][c] = gs.GetColor(&gs.World.Tiles[r][c])
		}
	}
}

func (gs *GameServer) CalculateMinimap() {
	topSnakes := gs.World.SortSnakes()[:gs.LeaderboardSize]
	var minimap []MinimapMessage // TODO Convert to minimapmessage

	for _, snake := range topSnakes {
		current := snake.Head
		for i := 0; i < snake.Length; i++ {
			minimap = append(minimap, MinimapMessage{
				Row:   current.Row,
				Col:   current.Col,
				Color: gs.GetColor(&gs.World.Tiles[current.Row][current.Col]),
			})
			current = current.Next
		}
	}
	gs.Minimap = minimap
}

func (gs *GameServer) PostGame() {
	// TODO token consumed
	// TODO Money awarded
}
