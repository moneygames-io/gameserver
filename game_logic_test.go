package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateMap(t *testing.T) {
	s := &State{
		SignupCount: 100,
		InitialConfig: &Config{
			ScalingFactor: 100,
		},
	}

	s.CreateMap()
	assert.Equal(t, len(s.World.Tiles), 1000)
	for _, row := range s.World.Tiles {
		assert.Equal(t, len(row), 1000)
	}
}

func TestState_FindRandomEmptyLocation(t *testing.T) {
	s := &State{
		SignupCount: 4,
		InitialConfig: &Config{
			ScalingFactor: 4,
		},
	}

	s.SetupLogger()
	s.SetRandomSeed()
	s.CreateMap()

	assert.Equal(t, 4, len(s.World.Tiles))
	for _, row := range s.World.Tiles {
		assert.Equal(t, 4, len(row))
	}

}
