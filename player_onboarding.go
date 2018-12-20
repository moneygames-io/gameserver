package main

import (
	"github.com/gorilla/websocket"
	"strings"
)

func (s *State) ValidateToken(token string) bool {
	status, _ := s.PlayerRedis.HGet(token, "status").Result()
	if status == "paid" {
		return true
	}
	return false
}

func (s *State) TokenConsumed(token string) {
	s.PlayerRedis.HSet(token, "status", "in game")
	s.PlayerRedis.HSet(token, "game", s.GameID)
}

func (s *State) NewConnectionHandler(conn *websocket.Conn) {
	s.Log.Info("New socket created")
	//err := conn.ReadJSON(&message)

	messageType, data, err := conn.ReadMessage()
	message := strings.Split(string(data), ",")

	if err != nil || messageType != websocket.BinaryMessage || len(message) < 1 {
		return
	}

	name := message[0]
	token := message[1]

	if token == "spectating" {
		s.SpectatorCount++
		s.Spectators[s.SpectatorCount] = &Spectator{
			Connection: conn,
			Name:       name,
		}
		return
	}

	if s.ValidateToken(token) {
		s.PlayerCount++
		s.TokenConsumed(token)
		newPlayer := &Player{
			Connection: conn,
			Token:      token,
			Name:       name,
			Input:      &Input{ZoomLevel: s.InitialConfig.DefaultZoom},
		}

		s.SpawnPlayer(newPlayer)
		go s.CollectInput(newPlayer)

		if s.PlayerCount == s.SignupCount && s.Running == false {
			s.StartGame()
		}
	}
}

func (s *State) SpawnPlayer(newPlayer *Player) {
	snake := &SnakeNode{}

	newPlayer.Snake = snake
	snake.Player = newPlayer
	s.World.ActivePlayers[newPlayer] = snake

	s.AddNewSnakeToWorld(snake)
}
