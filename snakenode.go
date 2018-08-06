package main

type SnakeNode struct {
	Snake *Snake
	Row   int
	Col   int
	Next  *SnakeNode
	Prev  *SnakeNode
}
