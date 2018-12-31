package main

import (
	"github.com/go-redis/redis"
	"github.com/op/go-logging"
	"github.com/pions/webrtc"
	"sync"
)

// Represents the state of the current game, root node of the object graph
// TODO move things that are not threadsafe to their own type?
// It's kinda strange that for some funcs we lock and for some we don't
// Could lock for everything for uniformity, but that feels strange as some functions only interact with logger/redis
// TokenConsumed highlights that a bunch of these aren't actually part of this program's state
type State struct {
	sync.Mutex

	// Used to identify "this" game in GameserverRedis
	GameID string

	// Used to provide status updates about "this" game's state
	GameserverRedis *redis.Client

	// Used to provide information related to players
	PlayerRedis *redis.Client

	// Logging
	Log *logging.Logger

	// Used to read configuration settings across the app
	InitialConfig *Config

	// How many players we're expecting
	SignupCount int

	// How many players have been spawned
	PlayerCount int

	// How many spectators are watching this game
	SpectatorCount int

	// The current frame rate of the game
	FrameRate int

	// Whether the game has started or not
	Running bool

	// 2D world which all the game logic operates on
	World *Map

	// Collection of spectators
	// TODO: Don't make this a map
	Spectators map[int]*Spectator

	// After a game loop has executed, these are the current rankings of the players
	Rankings []*Player
}

// Used to store all the information regarding a player
type Player struct {
	Name           string
	Token          string
	Snake          *SnakeNode
	SpectatorCount int
	Input          *Input
	Message        *Message
	Connection     *webrtc.RTCDataChannel
}

type Input struct {
	Direction int
	Sprinting bool
	ZoomLevel int
}

type Spectator struct {
	Name        string
	Connection  *webrtc.RTCDataChannel
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
	RTCSettings     webrtc.RTCConfiguration
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

// Program-wide constants
const FrameMessage = 1
const WonMessage = 2
const LostMessage = 3
