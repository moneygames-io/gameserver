package main

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	Name             string
	CurrentZoomLevel int
	Token            string
	Conn             *websocket.Conn
	Player           *Player
}

func NewClient(r *RegisterMessage, conn *websocket.Conn) *Client {
	c := &Client{}

	c.Name = r.Name
	c.Token = r.Token
	c.Conn = conn

	return c
}

func (c *Client) GetView(m *Map) [][]uint32 {
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

func (c *Client) CollectInput(conn *websocket.Conn) {
	msg := &ClientUpdateMessage{}
	for {
		conn.ReadJSON(msg)
		c.Player.CurrentDirection = msg.CurrentDirection
		c.Player.CurrentSprint = msg.CurrentSprint

		c.CurrentZoomLevel = msg.CurrentZoomLevel
	}
}
