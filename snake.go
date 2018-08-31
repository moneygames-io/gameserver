package main

import (
	"math"
)

type Snake struct {
	Player           *Player
	Length           int
	CurrentDirection int
	Head             *SnakeNode
	Tail             *SnakeNode
	Events           MapEvent
}

func NewSnake(row int, col int, events MapEvent, player ...*Player) *Snake {
	s := new(Snake)
	s.Head = &SnakeNode{s, row, col, nil, nil}
	s.Tail = s.Head
	s.Length = 1
	s.CurrentDirection = 0
	s.Events = events

	s.Events.SnakeCreated(s)
	if len(player) > 0 {
		s.Player = player[0]
	}
	return s
}

func (snake *Snake) Move(direction int) {
	if snake.Length == 1 || math.Abs(float64(direction-(snake.CurrentDirection))) != 2 {
		snake.CurrentDirection = direction
	}
	status := snake.GrowHead()

	switch status {
	case 0: // Empty Cell
		snake.ShortenTail(1)
		break
	case 1: // Found Food
		snake.Events.RemoveFood(snake.Head.Col, snake.Head.Row)
		break
	case 2: // Dead
		snake.Dead()
	}
}

func (snake *Snake) Sprint(direction int) {
	if snake.Length <= 4 {
		snake.Move(direction)
		return
	}

	if math.Abs(float64(direction-snake.CurrentDirection)) != 2 {
		snake.CurrentDirection = direction
	}

	status := snake.GrowHead() // TODO This is why sprint is not working

	switch status {
	case 0:
		oldTail := snake.ShortenTail(2)
		snake.Events.AddFood(&Food{oldTail.Row, oldTail.Col})
	case 1:
		oldTail := snake.ShortenTail(1)
		snake.Events.AddFood(&Food{oldTail.Row, oldTail.Col})
	case 2:
		snake.Dead()
	}
}

func (snake *Snake) GrowHead() int {
	oldHead := snake.Head
	dx, dy := directionToDxDy(snake.CurrentDirection)

	newHead := new(SnakeNode)
	newHead.Snake = oldHead.Snake
	newHead.Col = dx + oldHead.Col
	newHead.Row = dy + oldHead.Row
	newHead.Next = oldHead
	newHead.Prev = nil

	status := snake.Events.AddNode(newHead)

	if status != 2 {
		oldHead.Prev = newHead

		snake.Head = newHead
		snake.Length = snake.Length + 1
	}

	return status
}

func (snake *Snake) ShortenTail(howMuch int) *SnakeNode {
	if snake.Length == 1 {
		return nil
	}
	oldTail := snake.Tail
	newTail := snake.Tail.Prev

	newTail.Next = nil

	snake.Tail = newTail
	snake.Length = snake.Length - 1

	snake.Events.RemoveNode(oldTail.Row, oldTail.Col)

	if howMuch > 1 {
		return snake.ShortenTail(howMuch - 1)
	} else {
		return oldTail
	}
}

func (snake *Snake) Dead() {
	sn := snake.ShortenTail(1)
	if sn != nil {
		snake.Events.AddFood(&Food{sn.Row, sn.Col})
		snake.Dead()
	} else {
		snake.Length = 0
		snake.Events.RemoveNode(snake.Head.Row, snake.Head.Col)
		snake.Events.SnakeRemoved(snake)
		snake.Events.AddFood(&Food{snake.Head.Row, snake.Head.Col})
		snake.Tail = nil
		snake.Head = nil
	}
}

func directionToDxDy(direction int) (int, int) {
	switch direction {
	case 0:
		return 0, 1
	case 1:
		return 1, 0
	case 2:
		return 0, -1
	case 3:
		return -1, 0
	default:
		return 0, 0
	}
}
