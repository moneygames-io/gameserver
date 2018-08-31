package main

type LeaderboardMessage struct {
	Rank   int    `json:"Rank"`
	Name   string `json:"Name"`
	Length int    `json:"Length"`
	Color  uint32 `json: "Length"`
	Snake  *Snake `json:"-"`
}

func NewLeaderboardMessage(rank int, gs *GameServer, s *Snake) LeaderboardMessage {
	return LeaderboardMessage{
		Name:   s.Player.Client.Name,
		Length: s.Length,
		Color: gs.GetColor(&Tile{
			Snake: s,
			Food:  nil,
		}),
	}
}
