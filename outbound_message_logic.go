package main

import (
	"github.com/pions/webrtc/pkg/datachannel"
	"hash/fnv"
	"math"
	"reflect"
	"unsafe"
)

func (s *State) GenerateMessageModels() {
	for rank, player := range s.Rankings {

		// Rank & Spectator Count
		player.Message = &Message{}

		// Leaderboard & Minimap
		leaderboardSize := int(math.Min(float64(s.InitialConfig.LeaderboardSize), float64(len(s.Rankings))))

		if rank < leaderboardSize {
			player.Message.LeaderMap = s.Rankings[0:leaderboardSize]
		} else {
			player.Message.LeaderMap = append(s.Rankings[0:leaderboardSize], player)
		}

		// Perspective
		player.Message.Perspective = map[MapObject]bool{}
		player.Message.ViewportSize = player.Input.ZoomLevel * 2
		player.Message.MapSize = len(s.World.Tiles)

		p0Row := player.Snake.Row - player.Input.ZoomLevel
		p0Col := player.Snake.Col - player.Input.ZoomLevel

		player.Message.TopLeft = &Coordinate{p0Row, p0Col}

		for row := 0; row < player.Input.ZoomLevel*2; row++ {
			for col := 0; col < player.Input.ZoomLevel*2; col++ {

				coordinate := &Coordinate{row + p0Row, col + p0Col}

				switch v := s.World.Get(coordinate).(type) {
				case *SnakeNode:
					player.Message.Perspective[v] = true
					break
				case *FoodNode:
					player.Message.Perspective[v] = true
					break
				case *OutOfBounds:
				case nil:
					continue
				default:
					s.Log.Error("Unexpected Object in World")
				}
			}
		}
	}
}

func (s *State) SerializeMessages() {
	for _, player := range s.Rankings {

		player.Message.Serialized = []rune{
			FrameMessage,
			int32(player.Message.TopLeft.Row),
			int32(player.Message.TopLeft.Col),
			int32(player.Message.ViewportSize),
			int32(player.Message.MapSize),
		}

		for mo := range player.Message.Perspective {
			switch v := mo.(type) {
			case *SnakeNode:
				sn := []rune{hash(v.Player.Token), int32(v.Row), int32(v.Col)}
				player.Message.Serialized = append(player.Message.Serialized, sn...)
				break
			case *FoodNode:
				fn := []rune{'F', int32(v.Row), int32(v.Col)}
				player.Message.Serialized = append(player.Message.Serialized, fn...)
				break
			}
		}

		player.Message.Serialized = append(player.Message.Serialized, -1)

		for _, member := range player.Message.LeaderMap {
			player.Message.Serialized = append(player.Message.Serialized, []rune(member.Name)...)
			player.Message.Serialized = append(player.Message.Serialized, []rune{
				-2,
				int32(hash(player.Token)),
				int32(player.Snake.Row),
				int32(player.Snake.Col),
				int32(player.Snake.Length),
				int32(player.SpectatorCount),
			}...)
		}

		player.Message.Serialized = append(player.Message.Serialized, -4)
	}
}

func (s *State) SendMessagesToPlayers() {
	for _, player := range s.Rankings {
		header := *(*reflect.SliceHeader)(unsafe.Pointer(&player.Message.Serialized))
		header.Len *= 4
		header.Cap *= 4
		data := *(*[]byte)(unsafe.Pointer(&header))
		_ = player.Connection.Send(datachannel.PayloadBinary{Data: data})
	}
}

func (s *State) SendMessagesToSpectators() {
	for _, spectator := range s.Spectators {
		header := *(*reflect.SliceHeader)(unsafe.Pointer(&spectator.CurrentView.Message.Serialized))
		header.Len *= 4
		header.Cap *= 4
		data := *(*[]byte)(unsafe.Pointer(&header))
		_ = spectator.Connection.Send(datachannel.PayloadBinary{Data: data})
	}
}

func (s *State) SendWin(player *Player) {
	pot, _ := s.GameserverRedis.HGet(s.GameID, "pot").Result()
	s.PlayerRedis.HSet(player.Token, "status", "won")

	message := []rune{WonMessage}
	message = append(message, []rune(pot)...)
	message = append(message, -1)

	header := *(*reflect.SliceHeader)(unsafe.Pointer(&message))
	header.Len *= 4
	header.Cap *= 4
	data := *(*[]byte)(unsafe.Pointer(&header))
	_ = player.Connection.Send(datachannel.PayloadBinary{Data: data})
}

func (s *State) SendLoss(player *Player) {
	s.PlayerRedis.HSet(player.Token, "status", "won")

	message := []rune{LostMessage}

	header := *(*reflect.SliceHeader)(unsafe.Pointer(&message))
	header.Len *= 4
	header.Cap *= 4
	data := *(*[]byte)(unsafe.Pointer(&header))
	_ = player.Connection.Send(datachannel.PayloadBinary{Data: data})
}

func hash(s string) int32 { // TODO PRE-PRODUCTION Is this secure enough to use for the token?
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	hash := h.Sum32()
	i32 := int32(hash)
	if i32 < 0 {
		return -1 * i32
	}

	if i32 == 'F' {
		return i32 + 1
	}

	return i32
}
