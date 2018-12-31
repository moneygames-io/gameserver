package main

import (
	"github.com/op/go-logging"
	"github.com/pions/webrtc"
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
		RTCSettings: webrtc.RTCConfiguration{
			IceServers: []webrtc.RTCIceServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		},
	}
}

func (s *State) SetupMiscServerVariables() {
	// Redis stuff
	s.GameserverRedis = connectToRedis("redis-gameservers:6379", s.Log)
	s.PlayerRedis = connectToRedis("redis-players:6379", s.Log)

	// Which port am I bound too from docker swarm's perspective
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
	s.Log.Info("Broadcasting Idle")
	s.GameserverRedis.HSet(s.GameID, "status", "idle")
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
			s.GameserverRedis.HSet(s.GameID, "players", 0)
			return
		}
	}
}

func corsHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		} else {
			h(w, r)
		}
	}
}

// Sets up connection handler that dispatches a goroutine for every request
func (s *State) SetupConnectionHandler() {

	http.Handle("/player", corsHandler(s.NewPlayer))
	s.Log.Fatal(http.ListenAndServe(":10000", nil))
}
