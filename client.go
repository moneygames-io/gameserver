package main

import (
	"fmt"
	"github.com/gorilla/websocket"
)

type Client struct {
	Name             string
	CurrentZoomLevel int
	Token            string
	Conn             *websocket.Conn
	Player           *Player
	Spectator        bool
	Status           string
}

func NewClient(r *RegisterMessage, conn *websocket.Conn) *Client {
	c := &Client{}

	if r != nil {
		if r.Name == "" {
			c.Name = "unnamed"
		} else {
			c.Name = r.Name
		}

		c.Token = r.Token
	} else {
		c.Spectator = true
	}

	c.Conn = conn

	return c
}

func (c *Client) GetPerspective(gs *GameServer) [][]uint32 {
	m := gs.World
	head := c.Player.Snake.Head
	r0 := head.Row - c.CurrentZoomLevel
	c0 := head.Col - c.CurrentZoomLevel

	colors := make([][]uint32, c.CurrentZoomLevel*2)

	for i := range colors {
		colors[i] = make([]uint32, c.CurrentZoomLevel*2)
	}

	for row := 0; row < c.CurrentZoomLevel*2; row++ {
		for col := 0; col < c.CurrentZoomLevel*2; col++ {
			if row+r0 >= len(m.Tiles) ||
				row+r0 < 0 ||
				col+c0 >= len(m.Tiles[0]) ||
				col+c0 < 0 {
				colors[row][col] = 0xFFFFFF
			} else {
				colors[row][col] = gs.GetColor(&m.Tiles[row+r0][col+c0])
			}
		}
	}

	return colors
}

func (c *Client) GetMinimap(gs *GameServer) []MinimapMessage {
	genericMinimap := gs.Minimap
	current := c.Player.Snake.Head

	if gs.World.GetSnakeRank(c.Player.Snake) >= gs.LeaderboardSize {
		for i := 0; i < c.Player.Snake.Length; i++ {
			genericMinimap = append(genericMinimap, MinimapMessage{
				Row:   current.Row,
				Col:   current.Col,
				Color: gs.GetColor(&gs.World.Tiles[current.Row][current.Col]),
			})
			current = current.Next
		}
	}
	return genericMinimap

}

func (c *Client) GetLeaderboard(gs *GameServer) []LeaderboardMessage {
	clientSnake := c.Player.Snake
	rank := gs.World.GetSnakeRank(clientSnake)

	if rank >= gs.LeaderboardSize {
		return append(gs.Leaderboard[:gs.LeaderboardSize], NewLeaderboardMessage(rank, gs, clientSnake))
	} else {
		return gs.Leaderboard[:gs.LeaderboardSize]
	}
}

func (c *Client) CollectInput(conn *websocket.Conn) {
	if c.Spectator {
		return
	}
	msg := &ClientUpdateMessage{}
	for {
		err := conn.ReadJSON(msg)
		if err == nil {
			c.Player.CurrentDirection = msg.CurrentDirection
			c.Player.CurrentSprint = msg.CurrentSprint
			c.CurrentZoomLevel = msg.CurrentZoomLevel
		} else {
			fmt.Println("Error:", err)
			c.Player.Snake.Dead()
			return
		}
	}
}

func (c *Client) SendLeaderboard(gs *GameServer) {
	c.Conn.WriteJSON(map[string][]LeaderboardMessage{"Leaderboard": gs.Leaderboard[:gs.LeaderboardSize]})
}

func (c *Client) SendCustomLeaderboard(gs *GameServer) {
	c.Conn.WriteJSON(map[string][]LeaderboardMessage{"Leaderboard": c.GetLeaderboard(gs)})
}

type MinimapWrapper struct {
	Minimap []MinimapMessage
	Rows    int
	Cols    int
}

func (c *Client) SendCustomMinimap(gs *GameServer) {
	c.Conn.WriteJSON(map[string]MinimapWrapper{"Minimap": MinimapWrapper{
		Minimap: c.GetMinimap(gs),
		Rows:    len(gs.World.Tiles),
		Cols:    len(gs.World.Tiles[0]),
	}})
}

func (c *Client) SendMinimap(gs *GameServer) {
	c.Conn.WriteJSON(map[string]MinimapWrapper{"Minimap": MinimapWrapper{
		Minimap: gs.Minimap,
		Rows:    len(gs.World.Tiles),
		Cols:    len(gs.World.Tiles[0]),
	}})
	c.Conn.WriteJSON(map[string][]MinimapMessage{"Minimap": gs.Minimap})
}

func (c *Client) SendPerspective(gs *GameServer) {
	c.Conn.WriteJSON(map[string][][]uint32{"Perspective": c.GetPerspective(gs)})
}

func (c *Client) SendSpectatorView(gs *GameServer) {
	c.Conn.WriteJSON(map[string][][]uint32{"Perspective": gs.SpectatorView})
}

func (c *Client) SendStatus(gs *GameServer) {
	c.Conn.WriteJSON(map[string]string{"status": c.Status})
}

func (c *Client) SendPot(gs *GameServer) {
	c.Conn.WriteJSON(map[string]int{"pot": gs.Pot})
}
