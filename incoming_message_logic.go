package main

import (
	"github.com/pions/webrtc/pkg/datachannel"
)

// TODO Validate Sprint
// TODO Validate Zoom level
func (s *State) HandlePlayerInput(payload datachannel.Payload, player *Player) {
	s.Lock()
	defer s.Unlock()

	value := payload.(*datachannel.PayloadBinary).Data // TODO Can crash server

	player.Input.Direction = int(value[0])
	player.Input.ZoomLevel = int(value[1])
	player.Input.Sprinting = value[2] == 1
}

func (s *State) HandleSpectatorInput(payload datachannel.Payload, spectator *Spectator) {
	// TODO
}
