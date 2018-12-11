package main

import (
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	state := &State{}

	state.SetupLogger()
	state.SetupInitialConfig()
	state.SetRandomSeed()
	state.SetupMiscServerVariables()
	state.BroadcastState()
	state.SetSignupCount()
	state.CreateMap()
	state.SetupConnectionHandler()
}

func (s *State) SetupLogger() {
	s.Log = logging.MustGetLogger("Gameserver")

	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)

	logging.SetBackend(formatter)
}

func (s *State) SetupInitialConfig() {
	s.InitialConfig = &Config{
		ScalingFactor:   250,
		FoodPerPlayer:   100,
		SprintFactor:    2,
		LeaderboardSize: 2,
		FrameRate:       5,
		DefaultZoom:     10,
	}
}

func (s *State) SetupMiscServerVariables() {
	// Redis stuff
	s.GameserverRedis = connectToRedis("redis-gameservers:6379", s.Log)
	s.PlayerRedis = connectToRedis("redis-players:6379", s.Log)

	// Which port?
	id, present := os.LookupEnv("GSPORT")
	if present {
		s.GameID = id
		s.Log.Info("Intended Port: %v", s.GameID)
	} else {
		s.GameID = "10000"
		s.Log.Error("GameID not present, guessing: %v", s.GameID)
	}

	// Alloc variables
	s.Spectators = map[int]*Spectator{}

	// Initial Variable values
	s.Running = false

}

func (s *State) BroadcastState() {
	s.GameserverRedis.HSet(s.GameID, "status", "idle")
	s.Log.Info("Broadcasted Idle")
}

func (s *State) SetRandomSeed() {
	now := time.Now().UnixNano()
	rand.Seed(now)
	s.Log.Debug("Seed: %v", now)
}

func (s *State) SetSignupCount() {
	for {
		s.Log.Info("Checking for player count")
		playerCountString, _ := s.GameserverRedis.HGet(s.GameID, "players").Result()
		players, _ := strconv.Atoi(playerCountString)
		if players == 0 {
			s.Log.Debug("Player count unavailable, sleeping")
			time.Sleep(1000 * time.Millisecond)
		} else {
			s.GameserverRedis.HSet(s.GameID, "status", "ready")
			s.SignupCount = players
			s.Log.Info("Player Count: %v", players)
			return
		}
	}
}

func (s *State) SetupConnectionHandler() {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			http.Error(writer, "Could not create connection, please retry", http.StatusBadGateway)
			s.Log.Error("Could not create websocket")
			return
		}
		s.NewConnectionHandler(conn)
	})
	panic(http.ListenAndServe(":10000", nil))
}
