package main

func (s *State) SpawnFoodAtRandomLocation(howMuch int) {
	row, col := s.FindRandomEmptyLocation()
	fn := &FoodNode{row, col}
	s.World.Tiles[row][col] = fn

	if howMuch > 1 {
		s.SpawnFoodAtRandomLocation(howMuch - 1)
	}
}

func (s *State) SpawnFoodAtLocation(row, col int) {
	s.World.Tiles[row][col] = &FoodNode{row, col}
}
