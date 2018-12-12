package main

import (
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
)

type State struct {
	GameID          string
	GameserverRedis *redis.Client
	PlayerRedis     *redis.Client
	Log             *logging.Logger
	InitialConfig   *Config
	SignupCount     int
	PlayerCount     int
	SpectatorCount  int
	FrameRate       int
	Running         bool
	World           *Map
	Spectators      map[int]*Spectator // Don't make this a map
	Rankings        []*Player
}

type Player struct {
	Name           string
	Token          string
	Snake          *SnakeNode
	SpectatorCount int
	Input          *Input
	Message        *Message
	Connection     *websocket.Conn
}

type Input struct {
	Direction int
	Sprinting bool
	ZoomLevel int
}

type Spectator struct {
	Name        string
	Connection  *websocket.Conn
	CurrentView *Player
}

type Map struct {
	Tiles         [][]MapObject
	ActivePlayers map[*Player]*SnakeNode
	LostPlayers   map[*Player]*SnakeNode
}

type OutOfBounds struct{}

type MapObject interface{}

type SnakeNode struct {
	Player *Player
	Length int
	Next   *SnakeNode
	Row    int
	Col    int
}

type FoodNode struct {
	Row int
	Col int
}

type Config struct {
	ScalingFactor   int
	FoodPerPlayer   int
	SprintFactor    int
	LeaderboardSize int
	FrameRate       int
	DefaultZoom     int
}

type Message struct {
	TopLeft      *Coordinate
	ViewportSize int
	MapSize      int
	LeaderMap    []*Player
	Perspective  map[MapObject]bool
	Serialized   []int32
}

type Coordinate struct {
	Row int
	Col int
}

func (m *Map) Get(c *Coordinate) MapObject {
	row := c.Row
	col := c.Col

	if row < 0 || col < 0 || row >= len(m.Tiles) || col >= len(m.Tiles[0]) {
		return &OutOfBounds{}
	} else {
		return m.Tiles[row][col]
	}
}

const FrameMessage = 1
const WonMessage = 2
const LostMessage = 3
