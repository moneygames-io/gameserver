package main

import (
	"math/rand"
	"sort"
	"time"

	"github.com/Parth/boolean"
)

type Tile struct {
	Snake *Snake
	Food  *Food
}

type Map struct {
	Tiles         [][]Tile
	Players       map[*Player]*Snake
	Losers        map[*Player]*Snake
	FoodPerPlayer int
}

type MapEvent interface {
	SnakeCreated(*Snake)
	AddNode(*SnakeNode) int
	RemoveNode(int, int)
	AddFood(*Food)
	RemoveFood(int, int)
	SnakeRemoved(*Snake)
}

func NewMap(players int, scalingFactor int, foodFactor int) *Map {
	newMap := &Map{}
	newMap.Tiles = make([][]Tile, players*scalingFactor)
	newMap.FoodPerPlayer = foodFactor

	for i := range newMap.Tiles {
		newMap.Tiles[i] = make([]Tile, players*scalingFactor)
	}

	newMap.Players = make(map[*Player]*Snake)
	newMap.Losers = make(map[*Player]*Snake)
	return newMap
}

func (m *Map) SpawnNewPlayer(player *Player) (int, int) {
	rand.Seed(time.Now().UnixNano())
	row := rand.Intn(len(m.Tiles))
	col := rand.Intn(len(m.Tiles[0]))

	for m.Tiles[row][col].Snake != nil && m.Tiles[row][col].Food != nil {
		row = rand.Intn(len(m.Tiles))
		col = rand.Intn(len(m.Tiles[0]))
	}

	m.SpawnFood(m.FoodPerPlayer)

	m.SpawnNewPlayerAt(player, row, col)
	return row, col
}

func (m *Map) SpawnFood(num int) {
	rand.Seed(time.Now().UnixNano())
	row := rand.Intn(len(m.Tiles))
	col := rand.Intn(len(m.Tiles[0]))

	for m.Tiles[row][col].Snake != nil && m.Tiles[row][col].Food != nil {
		row = rand.Intn(len(m.Tiles))
		col = rand.Intn(len(m.Tiles[0]))
	}

	m.AddFood(&Food{row, col})

	if num-1 > 0 {
		m.SpawnFood(num - 1)
	}
}

func (m *Map) SpawnNewPlayerAt(player *Player, row int, col int) {
	m.Players[player] = NewSnake(row, col, m, player)
	player.Snake = m.Players[player]
}

func (m *Map) SnakeCreated(snake *Snake) {
	m.AddNode(snake.Head)
}

func (m *Map) AddNode(snakeNode *SnakeNode) int {
	col := snakeNode.Col
	row := snakeNode.Row

	if row >= len(m.Tiles) || col >= len(m.Tiles[0]) {
		return 2
	}

	if row < 0 || col < 0 {
		return 2
	}

	if m.Tiles[row][col].Snake != nil {
		if snakeNode.Snake != m.Tiles[row][col].Snake {
			return 2
		}
	}

	m.Tiles[row][col].Snake = snakeNode.Snake
	return boolean.BtoI(m.Tiles[row][col].Food != nil)
}

func (m *Map) RemoveNode(row int, col int) {
	m.Tiles[row][col].Snake = nil
}

func (m *Map) AddFood(food *Food) {
	col := food.Col
	row := food.Row

	m.Tiles[row][col].Food = food
}

func (m *Map) RemoveFood(col int, row int) {
	m.Tiles[row][col].Food = nil

}

func (m *Map) SnakeRemoved(snake *Snake) {
	m.Players[snake.Player] = nil
	delete(m.Players, snake.Player)
	m.Losers[snake.Player] = snake
}

func (m *Map) Update() {
	for player, snake := range m.Players {
		if player.CurrentSprint {
			snake.Sprint(player.CurrentDirection)
		} else {
			snake.Move(player.CurrentDirection)
		}
	}
}
