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
	World           *Map
	GameServerRedis *redis.Client
	PlayerRedis     *redis.Client
	ID              string
	GL              *gameLoop.GameLoop
	PlayerCount     int
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
		World:           NewMap(players, 30, 20),
		GameServerRedis: gameServerRedis,
		PlayerRedis:     playerRedis,
		ID:              id,
		PlayerCount:     players,
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

	if error != nil || !validateToken(message.Token, gs.PlayerRedis) {
		fmt.Println("Closing connection, token invalid", error, message)
		conn.Close()
	}

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
}

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
	leaderboard := gs.World.GetLeaderboard(len(gs.World.Players))

	for client := range gs.Users {
		var view [][]uint32

		if _, ok := gs.World.Losers[client.Player]; ok {
			view = gs.World.Render()
		} else {
			view = client.GetView(gs.World)
		}

		client.Conn.WriteJSON(map[string][][]uint32{"Perspective": view})
		client.Conn.WriteJSON(map[string][]string{"Leaderboard": leaderboard})
	}

	if len(gs.World.Players) == 1 {
		gs.PostGame()
		gs.PublishState("game finished")
		os.Exit(0)
	}
}

func (gs *GameServer) PostGame() {
	// TODO token consumed
	// TODO Money awarded
}
