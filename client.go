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
				colors[row][col] = m.GetColor(&m.Tiles[row+r0][col+c0])
			}
		}
	}

	return colors
}

func (c *Client) GetMinimap(m *Map) [][][]int {
	topSnakes := m.GetLeaderboardSnakes(10)
	var minimap [][]int

	for _, snake := range topSnakes {
		current := snake.Head
		for i := 0; i < snake.Length; i++ {
			minimap = append(minimap, []int{current.Row, current.Col})
			current = currrent.Next
		}
	}
}

func (c *Client) GetLeaderboard(gs *GameServer) []LeaderboardMessage {
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

func (c *Client) SendCustomLeaderboard(gs *GameServer) {
	c.Conn.WriteJSON(map[string][]LeaderboardMessage{"Leaderboard": c.GetLeaderboard(gs)})
}

func (c *Client) SendCustomMinimap(gs *GameServer) {
	c.Conn.WriteJSON(map[string][]MinimapMessage{"Minimap": c.GetMinimap(gs)})
}

func (c *Client) SendPerspective(gs *GameServer) {
	c.Conn.WriteJSON(map[string][][]uint32{"Perspective": c.GetPerspective(gs)})
}

func (c *Client) SendSpectatorView(gs *GameServer) {
	c.Conn.WriteJSON(map[string][][]uint32{"Perspective": gs.SpectatorView})
}

func (c *Client) SendMinimap(gs *GameServer) {
	c.Conn.WriteJSON(map[string][]MinimapMessage{"Minimap": gs.Minimap})
}
