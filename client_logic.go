package main

import (
	"github.com/gorilla/websocket"
)

// TODO Validate Sprint
// TODO Validate Zoom level
func (s *State) CollectInput(player *Player) {
	for {
		mType, value, err := player.Connection.ReadMessage()
		if err == nil && mType == websocket.BinaryMessage {

			player.Input.Direction = int(value[0])
			player.Input.ZoomLevel = int(value[1])
			player.Input.Sprinting = value[2] == 1

		} else {
			s.Log.Error("Socket Error: %v", err)
			// TODO Handle disconnect
			return
		}
	}
}
