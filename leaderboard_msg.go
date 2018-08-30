package main

type LeaderboardMessage struct {
	Name   string
	Length int
	Color  uint32
}

func NewLeaderboardMessage(m *Map, s *Snake) LeaderboardMessage {
	return LeaderboardMessage{
		Name:   s.Player.Client.Name,
		Length: s.Length,
		Color: m.GetColor(&Tile{
			Snake: s,
			Food:  nil,
		}),
	}
}
