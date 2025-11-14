package internal

import (
	"testing"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
)

func TestGenerateRandomLayout(t *testing.T) {
	if len(allowedWallSpaces) == 0 {
		t.Errorf("Expected allowedWallSpaces to not be empty, got len() 0")
	}

	testBoard := Board{
		Entities: make(map[Location]Entity),
	}

	testBoard.GenerateRandomLayout()

	if len(testBoard.Entities) == 0 {
		t.Errorf("Expected walls to be generated")
	}

	for _, value := range testBoard.Entities {
		if value.Type != models.EntityType_ENTITY_TYPE_WALL {
			t.Errorf("non wall entity generated")
		}

		if value.PlayerID != NeutralID {
			t.Errorf("non nuetral id found in generated wall")
		}
	}
}

func TestPregeneratedlayout(t *testing.T) {
	// Before initializing, it will be nil
	if PreMadeLayouts != nil {
		t.Errorf("non nil value of PreMadeLayouts before initializing")
	}

	err := InitializePremadelayouts("./../images")
	if err != nil {
		t.Errorf("errored while initializing premade layouts")
	}

	if len(PreMadeLayouts) <= 0 {
		t.Errorf("premade layouts should have some layouts in it")
	}

	// We know one of the sample premade layouts is the arena
	found := false
	walls := []Location{}
	for _, currentLayout := range PreMadeLayouts {
		if currentLayout.Name == "arena.png" {
			found = true
			walls = currentLayout.Walls
		}
	}

	if found != true {
		t.Errorf("did not find arena layout")
	}

	// target some locations we know will be walls
	knownLocations := []Location{
		{
			X: 0,
			Y: 0,
		},
		{
			X: 5,
			Y: 6,
		},
		{
			X: 10,
			Y: 10,
		},
	}

	for _, currentLocation := range knownLocations {
		foundWall := false
		for _, curentWall := range walls {
			if currentLocation.X == curentWall.X && currentLocation.Y == curentWall.Y {
				foundWall = true
			}
		}

		if !foundWall {
			t.Errorf("did not find wall %d,%d in arena layout", currentLocation.X, currentLocation.Y)
		}
	}

}
