package main

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

// Creates the map based on how many players are destined to join and scaling factor
func (s *State) CreateMap() {
	s.World = &Map{}

	// Initialize Player Categories
	s.World.ActivePlayers = map[*Player]*SnakeNode{}
	s.World.LostPlayers = map[*Player]*SnakeNode{}

	// Compute Map Bounds
	mapSize := int(math.Sqrt(float64(s.SignupCount))) * s.InitialConfig.ScalingFactor

	// Initialize Map
	s.World.Tiles = make([][]MapObject, mapSize)
	for i := range s.World.Tiles {
		s.World.Tiles[i] = make([]MapObject, mapSize)
	}

	s.Log.Info("Created an %v x %v map", mapSize, mapSize)
}

func (s *State) StartGame() {

	s.Lock()
	s.Log.Info("Game started")
	s.Running = true
	s.FrameRate = s.InitialConfig.FrameRate
	s.Unlock()

	s.FrameUpdater()
}

func (s *State) FindRandomEmptyLocation() (int, int) {
	row := rand.Intn(len(s.World.Tiles))
	col := rand.Intn(len(s.World.Tiles[0]))

	if s.World.Get(&Coordinate{row, col}) != nil {
		return s.FindRandomEmptyLocation()
	}

	return row, col
}

func (s *State) IsLocationInBounds(row int, col int) bool {
	return row > 0 && col > 0 && row < len(s.World.Tiles) && col < len(s.World.Tiles[0])
}

func (s *State) FrameUpdater() {
	for s.Running && len(s.World.ActivePlayers) > 1 {
		s.Log.Info("Current Framerate: %v", s.FrameRate)

		startTime := time.Now()

		s.Lock()
		s.MoveSnakesForward()
		s.CalculateRankings()
		s.GenerateMessageModels()
		s.SerializeMessages()
		s.SendMessagesToPlayers()
		s.SendMessagesToSpectators()
		s.Unlock()
		rate := time.Second / time.Duration(s.FrameRate)
		delta := time.Since(startTime)

		sleepAmount := rate.Nanoseconds() - delta.Nanoseconds()

		s.Log.Debug("Frame Delta: %v, Sleep Time: %v", delta, time.Duration(sleepAmount))

		if sleepAmount > 0 { // We have extra time to sleep
			time.Sleep(time.Duration(sleepAmount))
		}
	}

	s.SendWin(s.Rankings[0])
}

func (s *State) CalculateRankings() {
	players := make([]*Player, len(s.World.ActivePlayers))

	index := 0
	for k := range s.World.ActivePlayers {
		players[index] = k
		index++
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].Snake.Length > players[j].Snake.Length
	})

	s.Rankings = players
}

func (s *State) MoveSnakesForward() {
	s.Log.Info("Moving %v snakes forward", len(s.World.ActivePlayers))
	for player, snake := range s.World.ActivePlayers { // TODO what if ActivePlayers changes?
		if player.Input.Sprinting {
			s.Sprint(snake)
		} else {
			s.Move(snake)
		}
	}
}
