package main

import (
	"fmt"
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
	SpectatorView   [][]uint32
	Leaderboard     []LeaderboardMessage
	Spectators      []*Client
	Minimap         []MinimapMessage
}

var gameserver *GameServer

func main() {
	gameServerRedis := connectToRedis("redis-gameservers:6379")
	playerRedis := connectToRedis("redis-players:6379")
	id := os.Getenv("GSPORT")

	gameServerRedis.HSet(id, "status", "idle")

	players := getPlayers(id, gameServerRedis)

	gameserver = &GameServer{
		Users:           make(map[*Client]*Player),
		Colors:          make(map[*Snake]uint32),
		World:           NewMap(players, 30, 20),
		GameServerRedis: gameServerRedis,
		PlayerRedis:     playerRedis,
		ID:              id,
		PlayerCount:     players,
	}

	gameserver.SpectatorView = make([][]uint32, len(gs.World.Tiles))

	for i := range gs.World.Tiles {
		gameserver.SpectatorView[i] = make([]uint32, len(gs.World.Tiles[i]))
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
		c.Player = &Player{}
		gs.World.SpawnNewPlayer(c.Player)
		c.Player.Client = c

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
	gs.CalculateSpectatorView()
	gs.CalculateLeaderboard(10)

	for player := range gs.World.Players {
		player.client.SendPerspective(gs)
		player.client.SendCustomLeaderboard(gs)
		player.Client.SendCustomMinimap(gs)
	}

	for loser := range gs.World.Loser {
		loser.client.SendSpectatorView(gs)
		loser.client.SendCustomLeaderboard(gs)
		loser.client.SendMinimap(gs)
	}

	for index, spectator := range gs.spectators {
		spectator.SendSpectatorView(gs)
		spectator.SendLeaderboard(gs)
		spectator.SendMinimap(gs)
	}

	if len(gs.World.Players) == 1 {
		gs.PostGame()
		gs.PublishState("game finished")
		os.Exit(0)
	}
}

func (gs *GameServer) GetTopSnakes(topN int, m *Map) []*Snake {
	if topN < len(m.Players) {
		topN = len(m.Players)
	}

	snakes := make([]*Snake, len(m.Players))

	index := 0
	for _, v := range m.Players {
		snakes[index] = v
		index++
	}

	sort.Slice(snakes, func(i, j int) bool {
		return snakes[i].Length < snakes[j].Length
	})

	return snakes[0:topN]
}

func (gs *GameServer) CalculateLeaderboard(topN int) {
	leaderboard := make([]LeaderboardMessage, topN)
	snakes := m.GetTopSnakes(topN)
	for i, snake := range snakes {
		leaderboard[i] = NewLeaderboardMessage(i, m, snake)
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

func (gs *GameServer) CalculateSpectatorView() [][]uint32 {
	for r := range gs.World.Tiles {
		for c := range gs.World.Tiles[r] {
			gs.SpectatorView[r][c] = gs.World.GetColor(&gs.World.Tiles[r][c])
		}
	}

	return colors
}

func (gs *GameServer) CalculateMinimap(int topN) []MinimapMessage {
	topSnakes := m.GetTopSnakes(10)
	var minimap []MinimapMessage // TODO Convert to minimapmessage

	for _, snake := range topSnakes {
		current := snake.Head
		for i := 0; i < snake.Length; i++ {
			minimap = append(minimap, MinimapMessage{
				Row: current.Row,
				Col: current.Col,
				Color: gs.GetColor(gs.World.Tiles[current.Row][current.Col])
			})
			current = currrent.Next
		}
	}
	gs.Minimap = minimap
}

func (gs *GameServer) PostGame() {
	// TODO token consumed
	// TODO Money awarded
}
