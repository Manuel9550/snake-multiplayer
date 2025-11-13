package internal

import (
	"testing"
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

}
