package main

type Player struct {
	CurrentDirection int
	CurrentSprint    bool
	Snake            *Snake
	Client           *Client
}
