package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/datachannel"
	"github.com/pions/webrtc/pkg/ice"
	"net/http"
	"strconv"
)

// Entry point for a players
// Validates Request
// Sets up offer
// Listens for RTCDataChannel
// http's goroutine
func (s *State) NewPlayer(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	if request.Body == nil {
		http.Error(writer, "No Body", 400)
		return
	}

	input := map[string]string{}
	err := json.NewDecoder(request.Body).Decode(&input)

	if err != nil {
		http.Error(writer, err.Error(), 400)
		return
	}

	if input["token"] == "" {
		http.Error(writer, "No token", 400)
		return
	}

	if input["offer"] == "" {
		http.Error(writer, "No offer", 400)
		return
	}

	if !s.tokenIsValid(input["token"]) {
		http.Error(writer, "Invalid Token", 400)
		return
	}

	newPlayer := &Player{
		Token: input["token"],
		Name:  input["name"],
		Input: &Input{ZoomLevel: s.InitialConfig.DefaultZoom},
	}

	answer, err := s.SetupRTCForPlayer(newPlayer, input["offer"])
	if err != nil {
		http.Error(writer, "Could not create response", 500)
	}

	writer.WriteHeader(200)
	_, _ = writer.Write([]byte(answer))
}

func (s *State) SetupRTCForPlayer(player *Player, offer string) (string, error) {
	peerConnection, err := webrtc.New(s.InitialConfig.RTCSettings)
	if err != nil {
		return "", err
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState ice.ConnectionState) { // TODO this handles disconnects
		s.Log.Info("ICE Connection State has changed: %s\n", connectionState.String())
	})

	peerConnection.OnDataChannel(func(d *webrtc.RTCDataChannel) { // Called on a fresh goroutine (from the one for DC's)

		// goroutine
		d.OnOpen(func() { // Called from the DC listen goroutine
			s.OnBoardPlayer(player, d)
		})

		d.OnMessage(func(payload datachannel.Payload) { // Called from third goroutine
			s.HandlePlayerInput(payload, player)
		})
	})

	return s.CreateAnswer(offer, peerConnection) // Sets up a goroutine to listen for DC Connects
}

func (s *State) CreateAnswer(offer64 string, peerConnection *webrtc.RTCPeerConnection) (string, error) {
	jsonOffer, err := base64.StdEncoding.DecodeString(offer64)
	if err != nil {
		return "", err
	}

	offer := webrtc.RTCSessionDescription{}
	err = json.Unmarshal(jsonOffer, &offer)
	if err != nil {
		return "", err
	}

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		return "", err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	jsonAnswer, err := json.Marshal(answer)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonAnswer), nil
}

func (s *State) OnBoardPlayer(p *Player, d *webrtc.RTCDataChannel) {
	s.Lock()
	defer s.Unlock()

	p.Connection = d
	s.SpawnPlayer(p)
	s.PlayerCount++
	s.TokenConsumed(p.Token)
	if s.PlayerCount == s.SignupCount && s.Running == false {
		go func() {
			s.StartGame()
		}()
	}
}

func (s *State) SpawnPlayer(newPlayer *Player) {
	snake := &SnakeNode{}

	newPlayer.Snake = snake
	snake.Player = newPlayer
	s.World.ActivePlayers[newPlayer] = snake

	s.AddNewSnakeToWorld(snake)
}

func (s *State) NewSpectator(writer http.ResponseWriter, request *http.Request) {
	if request.Body == nil {
		http.Error(writer, "No Body", 400)
		return
	}

	input := map[string]string{}
	err := json.NewDecoder(request.Body).Decode(&input)

	if err == nil {
		http.Error(writer, err.Error(), 400)
		return
	}

	if input["offer"] == "" {
		http.Error(writer, "No offer", 400)
		return
	}

	newSpectator := &Spectator{
		Name: input["name"],
	}

	answer, err := s.SetupRTCForSpectator(newSpectator, input["offer"])
	if err != nil {
		http.Error(writer, "Could not create response", 500)
	}

	writer.WriteHeader(200)
	_, _ = writer.Write([]byte(answer))
}

func (s *State) SetupRTCForSpectator(spectator *Spectator, offer string) (string, error) {
	peerConnection, err := webrtc.New(s.InitialConfig.RTCSettings)
	if err != nil {
		return "", err
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState ice.ConnectionState) { // TODO this handles disconnects
		s.Log.Info("ICE Connection State has changed: %s\n", connectionState.String())
	})

	peerConnection.OnDataChannel(func(d *webrtc.RTCDataChannel) { // Called on a fresh goroutine (from the one for DC's)

		// goroutine
		d.OnOpen(func() { // Called from the DC listen goroutine
			s.OnBoardSpectator(spectator, d)
		})

		d.OnMessage(func(payload datachannel.Payload) { // Called from third goroutine
			s.HandleSpectatorInput(payload, spectator)
		})
	})

	return s.CreateAnswer(offer, peerConnection)
}

func (s *State) OnBoardSpectator(spectator *Spectator, datachannel *webrtc.RTCDataChannel) {

}

func (s *State) tokenIsValid(token string) bool {
	status, _ := s.PlayerRedis.HGet(token, "status").Result()
	if status == "paid" {
		return true
	}
	return false
}

func (s *State) TokenConsumed(token string) {
	unconfirmed, _ := s.PlayerRedis.HGet(token, "unconfirmed").Result()
	incr, _ := strconv.ParseInt(unconfirmed, 10, 64) // incr must be base 10 int64
	s.GameserverRedis.HIncrBy(s.GameID, "unconfirmed", incr)
	s.PlayerRedis.HSet(token, "status", "in game")
	s.PlayerRedis.HSet(token, "game", s.GameID)
	s.PlayerRedis.SAdd(s.GameID, token)
}
