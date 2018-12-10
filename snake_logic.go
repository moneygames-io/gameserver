package main

func (s *State) AddNewSnakeToWorld(sn *SnakeNode) {
	sn.Row, sn.Col = s.FindRandomEmptyLocation()
	sn.Length = 1

	s.World.Tiles[sn.Row][sn.Col] = sn
	s.SpawnFoodAtRandomLocation(s.InitialConfig.FoodPerPlayer)
}

func (s *State) Sprint(snake *SnakeNode) {
	for i := 0; i < s.InitialConfig.SprintFactor; i++ {
		s.Move(snake)
	}
}

func (s *State) Move(snake *SnakeNode) {
	dRow, dCol := directionToRowCol(snake.Player.Input.Direction)

	newRow := snake.Row + dRow
	newCol := snake.Col + dCol

	coord := &Coordinate{newRow, newCol}

	// New head that gets added as long as this snake isn't dying
	newHead := &SnakeNode{
		Row:    newRow,
		Col:    newCol,
		Player: snake.Player,
		Next:   snake,
	}

	switch s.World.Get(coord).(type) {
	case *SnakeNode:
		s.Dead(snake)
		return
	case *FoodNode:
		// Set Length
		newHead.Length = snake.Length + 1

		// Iterate through and update Length
		tempSnake := snake
		for tempSnake != nil {
			tempSnake.Length = newHead.Length
			tempSnake = tempSnake.Next
		}
		break

	case *OutOfBounds:
		s.Dead(snake)
		return
	case nil:
		// Set Length
		newHead.Length = snake.Length

		// Iterate to tail
		tempSnake := newHead
		for tempSnake.Next.Next != nil {
			tempSnake = tempSnake.Next
		}

		// Remove it
		nodeToRemove := tempSnake.Next
		tempSnake.Next = nil
		s.World.Tiles[nodeToRemove.Row][nodeToRemove.Col] = nil
		break
	default:
		s.Log.Error("Unexpected Object in World")
	}

	// Make Rest of World aware of new head
	player := newHead.Player
	player.Snake = newHead
	s.World.ActivePlayers[player] = newHead
	s.World.Tiles[newRow][newCol] = newHead
}

func (s *State) Dead(snake *SnakeNode) {
	lastHead := snake
	player := lastHead.Player

	s.World.ActivePlayers[player] = nil
	s.World.LostPlayers[player] = lastHead
	delete(s.World.ActivePlayers, player)

	tempSN := lastHead
	for tempSN != nil {
		s.SpawnFoodAtLocation(tempSN.Row, tempSN.Col)
		tempSN = tempSN.Next
	}

	lastHead.Next = nil

	s.SendLoss(player)
}

func directionToRowCol(direction int) (int, int) {
	switch direction {
	case 0:
		return -1, 0
	case 1:
		return 0, 1
	case 2:
		return 1, 0
	case 3:
		return 0, -1
	default:
		return 0, 0
	}
}
