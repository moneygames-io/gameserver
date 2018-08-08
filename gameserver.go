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
	Users       map[*Client]*Player
	World       *Map
	RedisClient *redis.Client
	ID          string
	GL          *gameLoop.GameLoop
	PlayerCount int
}

var gameserver *GameServer

func main() {
	redisClient := connectToRedis()
	id := os.Getenv("GSPORT")

	players := 3

	fmt.Println(players)

	gameserver = &GameServer{
		Users:       make(map[*Client]*Player),
		World:       NewMap(2),
		RedisClient: redisClient,
		ID:          id,
		PlayerCount: players,
	}

	gameserver.GL = gameLoop.New(5, gameserver.MapUpdater)

	// TODO set status to ready
	http.HandleFunc("/ws", wsHandler)
	panic(http.ListenAndServe(":10000", nil))

}

func getPlayers(id string, redisClient *redis.Client) int {
	for {
		playerCountString, _ := redisClient.Get(id).Result()
		players, _ := strconv.Atoi(playerCountString)
		if players == 0 {
			time.Sleep(1000 * time.Millisecond)
			fmt.Println("No players yet")
		} else {
			return players
		}
	}

	return 0
}

func connectToRedis() *redis.Client {
	var client *redis.Client
	for {
		client = redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
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

	fmt.Println("gameserver connected to redis")

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

	if error != nil || !validateToken(message.Token) {
		conn.Close()
	}

	// TODO token consumed

	c := NewClient(message, conn)
	c.Player = &Player{}
	gs.World.SpawnNewPlayer(c.Player)

	gs.Users[c] = c.Player
	go c.CollectInput(conn)

	fmt.Println(len(gs.Users), gs.PlayerCount)
	if len(gs.Users) >= gs.PlayerCount && gs.GL.Running == false {
		gs.GL.Start()
		fmt.Println("started")
	}
}

func validateToken(token string) bool {
	// TODO validate token
	return true
}

func (gs *GameServer) PublishState(msg string) {
	gs.RedisClient.Set(gs.ID, msg, 0)
}

func (gs *GameServer) MapUpdater(delta float64) {
	gs.PublishState("game started")
	gs.World.Update()

	for client := range gs.Users {
		var view [][]uint32

		if _, ok := gs.World.Losers[client.Player]; ok {
			view = gs.World.Render()
		} else {
			view = client.GetView(gs.World)
			fmt.Println(view)
		}

		go client.Conn.WriteJSON(&view)
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
