package internal

import (
	"testing"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
)

// TestQueueEnqueue tests adding items to the queue
func TestQueueEnqueue(t *testing.T) {
	q := Queue{}

	q.Enqueue(Location{X: 1, Y: 2})
	q.Enqueue(Location{X: 3, Y: 4})

	if q.Len() != 2 {
		t.Errorf("Expected queue length 2, got %d", q.Len())
	}

	first := q.Get(0)
	if first == nil || first.X != 1 || first.Y != 2 {
		if first == nil {
			t.Error("Expected first element to exist")
		} else {
			t.Errorf("Expected first element {1, 2}, got {%d, %d}", first.X, first.Y)
		}
	}

	second := q.Get(1)
	if second == nil || second.X != 3 || second.Y != 4 {
		if second == nil {
			t.Error("Expected second element to exist")
		} else {
			t.Errorf("Expected second element {3, 4}, got {%d, %d}", second.X, second.Y)
		}
	}
}

// TestQueueDequeue tests removing items from the queue
func TestQueueDequeue(t *testing.T) {
	q := Queue{}
	q.Enqueue(Location{X: 1, Y: 2})
	q.Enqueue(Location{X: 3, Y: 4})

	item := q.Dequeue()
	if item == nil {
		t.Fatal("Expected non-nil item from Dequeue")
	}

	if item.X != 1 || item.Y != 2 {
		t.Errorf("Expected dequeued element {1, 2}, got {%d, %d}", item.X, item.Y)
	}

	if q.Len() != 1 {
		t.Errorf("Expected queue length 1 after dequeue, got %d", q.Len())
	}

	item = q.Dequeue()
	if item == nil {
		t.Fatal("Expected non-nil item from second Dequeue")
	}

	if item.X != 3 || item.Y != 4 {
		t.Errorf("Expected dequeued element {3, 4}, got {%d, %d}", item.X, item.Y)
	}

	if q.Len() != 0 {
		t.Errorf("Expected empty queue after two dequeues, got length %d", q.Len())
	}
}

// TestQueueDequeueEmpty tests dequeuing from an empty queue
func TestQueueDequeueEmpty(t *testing.T) {
	q := Queue{}

	item := q.Dequeue()
	if item != nil {
		t.Error("Expected nil when dequeuing from empty queue")
	}
}

// TestQueueIsEmpty tests the IsEmpty method
func TestQueueIsEmpty(t *testing.T) {
	q := Queue{}

	if !q.IsEmpty() {
		t.Error("Expected queue to be empty initially")
	}

	q.Enqueue(Location{X: 1, Y: 2})

	if q.IsEmpty() {
		t.Error("Expected queue to not be empty after enqueue")
	}

	q.Dequeue()

	if !q.IsEmpty() {
		t.Error("Expected queue to be empty after dequeuing all items")
	}
}

// TestQueueToProto tests converting queue to proto format
func TestQueueToProto(t *testing.T) {
	q := Queue{}
	q.Enqueue(Location{X: 5, Y: 10})
	q.Enqueue(Location{X: 15, Y: 20})

	protoFields := q.ToProto()

	if len(protoFields) != 2 {
		t.Errorf("Expected 2 proto fields, got %d", len(protoFields))
	}

	if protoFields[0].X != 5 || protoFields[0].Y != 10 {
		t.Errorf("Expected first field {5, 10}, got {%d, %d}", protoFields[0].X, protoFields[0].Y)
	}

	if protoFields[1].X != 15 || protoFields[1].Y != 20 {
		t.Errorf("Expected second field {15, 20}, got {%d, %d}", protoFields[1].X, protoFields[1].Y)
	}
}

// TestQueueToProtoEmpty tests converting empty queue to proto format
func TestQueueToProtoEmpty(t *testing.T) {
	q := Queue{}

	protoFields := q.ToProto()

	if len(protoFields) != 0 {
		t.Errorf("Expected 0 proto fields for empty queue, got %d", len(protoFields))
	}
}

// TestUpdateDirectionValid tests valid direction updates
func TestUpdateDirectionValid(t *testing.T) {
	tests := []struct {
		name         string
		initial      models.Direction
		newDirection models.Direction
		expected     models.Direction
	}{
		{"Right to Up", models.Direction_DIRECTION_RIGHT, models.Direction_DIRECTION_UP, models.Direction_DIRECTION_UP},
		{"Right to Down", models.Direction_DIRECTION_RIGHT, models.Direction_DIRECTION_DOWN, models.Direction_DIRECTION_DOWN},
		{"Up to Left", models.Direction_DIRECTION_UP, models.Direction_DIRECTION_LEFT, models.Direction_DIRECTION_LEFT},
		{"Up to Right", models.Direction_DIRECTION_UP, models.Direction_DIRECTION_RIGHT, models.Direction_DIRECTION_RIGHT},
		{"Down to Left", models.Direction_DIRECTION_DOWN, models.Direction_DIRECTION_LEFT, models.Direction_DIRECTION_LEFT},
		{"Down to Right", models.Direction_DIRECTION_DOWN, models.Direction_DIRECTION_RIGHT, models.Direction_DIRECTION_RIGHT},
		{"Left to Up", models.Direction_DIRECTION_LEFT, models.Direction_DIRECTION_UP, models.Direction_DIRECTION_UP},
		{"Left to Down", models.Direction_DIRECTION_LEFT, models.Direction_DIRECTION_DOWN, models.Direction_DIRECTION_DOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &Player{
				Direction: tt.initial,
			}
			player.UpdateDirection(tt.newDirection)

			if player.Direction != tt.expected {
				t.Errorf("Expected direction %v, got %v", tt.expected, player.Direction)
			}
		})
	}
}

// TestUpdateDirectionInvalid tests that invalid direction changes are rejected
func TestUpdateDirectionInvalid(t *testing.T) {
	tests := []struct {
		name         string
		initial      models.Direction
		newDirection models.Direction
		expected     models.Direction
	}{
		{"Up to Down", models.Direction_DIRECTION_UP, models.Direction_DIRECTION_DOWN, models.Direction_DIRECTION_UP},
		{"Down to Up", models.Direction_DIRECTION_DOWN, models.Direction_DIRECTION_UP, models.Direction_DIRECTION_DOWN},
		{"Left to Right", models.Direction_DIRECTION_LEFT, models.Direction_DIRECTION_RIGHT, models.Direction_DIRECTION_LEFT},
		{"Right to Left", models.Direction_DIRECTION_RIGHT, models.Direction_DIRECTION_LEFT, models.Direction_DIRECTION_RIGHT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &Player{
				Direction: tt.initial,
			}
			player.UpdateDirection(tt.newDirection)

			if player.Direction != tt.expected {
				t.Errorf("Expected direction to remain %v, got %v", tt.expected, player.Direction)
			}
		})
	}
}

// TestUpdateState tests updating player state
func TestUpdateState(t *testing.T) {
	player := &Player{
		State: models.PlayerState_STATE_CONNECTING,
	}

	player.UpdateState(models.PlayerState_STATE_CONNECTED)
	if player.State != models.PlayerState_STATE_CONNECTED {
		t.Errorf("Expected state STATE_CONNECTED, got %v", player.State)
	}

	player.UpdateState(models.PlayerState_STATE_READY)
	if player.State != models.PlayerState_STATE_READY {
		t.Errorf("Expected state STATE_READY, got %v", player.State)
	}

	player.UpdateState(models.PlayerState_STATE_DONE)
	if player.State != models.PlayerState_STATE_DONE {
		t.Errorf("Expected state STATE_DONE, got %v", player.State)
	}
}

// TestToPlayerProtoMessage tests converting player to proto message
func TestToPlayerProtoMessage(t *testing.T) {
	bodyQueue := Queue{}
	bodyQueue.Enqueue(Location{X: 5, Y: 10})
	bodyQueue.Enqueue(Location{X: 6, Y: 10})

	player := &Player{
		ID:        "test-player-123",
		X:         10,
		Y:         20,
		Direction: models.Direction_DIRECTION_RIGHT,
		Colour:    "blue",
		BodyQueue: bodyQueue,
	}

	proto := player.ToPlayerProtoMessage()

	if proto.PlayerID != "test-player-123" {
		t.Errorf("Expected PlayerID 'test-player-123', got %s", proto.PlayerID)
	}

	if proto.Head.X != 10 || proto.Head.Y != 20 {
		t.Errorf("Expected head position {10, 20}, got {%d, %d}", proto.Head.X, proto.Head.Y)
	}

	if proto.PlayerDirection != models.Direction_DIRECTION_RIGHT {
		t.Errorf("Expected direction RIGHT, got %v", proto.PlayerDirection)
	}

	if proto.Colour != "blue" {
		t.Errorf("Expected colour 'blue', got %s", proto.Colour)
	}

	if len(proto.Body) != 2 {
		t.Errorf("Expected body length 2, got %d", len(proto.Body))
	}
}

// TestNewPlayerSlots tests initialization of player slots
func TestNewPlayerSlots(t *testing.T) {
	slots := NewPlayerSlots()

	if len(slots.Slots) != 4 {
		t.Errorf("Expected 4 slots, got %d", len(slots.Slots))
	}

	expectedColors := []string{"blue", "green", "red", "yellow"}
	for i, slot := range slots.Slots {
		if slot.Player != nil {
			t.Errorf("Expected slot %d to be empty, but has a player", i)
		}
		if slot.Colour != expectedColors[i] {
			t.Errorf("Expected slot %d to have colour %s, got %s", i, expectedColors[i], slot.Colour)
		}
	}
}

// TestPlayerSlotsCanJoin tests the CanJoin method
func TestPlayerSlotsCanJoin(t *testing.T) {
	slots := NewPlayerSlots()

	// Initially all slots are empty, should be able to join
	if !slots.CanJoin() {
		t.Error("Expected CanJoin to be true with empty slots")
	}

	// Add players to fill slots
	for i := 0; i < 4; i++ {
		player := &Player{
			ID:     "player-" + string(rune('1'+i)),
			Colour: "test",
		}
		success := slots.AddPlayer(player)
		if !success {
			t.Fatalf("Failed to add player %d", i)
		}
	}

	// All slots full, should not be able to join
	if slots.CanJoin() {
		t.Error("Expected CanJoin to be false with all slots full")
	}
}

// TestPlayerSlotsAddPlayer tests adding players to slots
func TestPlayerSlotsAddPlayer(t *testing.T) {
	slots := NewPlayerSlots()

	player1 := &Player{
		ID:     "player-1",
		Colour: "white",
	}

	// Add first player
	success := slots.AddPlayer(player1)
	if !success {
		t.Error("Expected AddPlayer to succeed for first player")
	}

	// Verify player was added to first slot
	if slots.Slots[0].Player != player1 {
		t.Error("Expected player to be in first slot")
	}

	// Add three more players
	for i := 2; i <= 4; i++ {
		player := &Player{
			ID:     "player-" + string(rune('0'+i)),
			Colour: "white",
		}
		success := slots.AddPlayer(player)
		if !success {
			t.Errorf("Expected AddPlayer to succeed for player %d", i)
		}
	}

	// Try to add fifth player (should fail)
	player5 := &Player{
		ID:     "player-5",
		Colour: "white",
	}
	success = slots.AddPlayer(player5)
	if success {
		t.Error("Expected AddPlayer to fail when all slots are full")
	}
}

// TestPlayerSlotsRemovePlayer tests removing players from slots
func TestPlayerSlotsRemovePlayer(t *testing.T) {
	slots := NewPlayerSlots()

	// Add a player
	player := &Player{
		ID:     "player-1",
		Colour: "white",
		conn:   nil, // No connection for testing
	}
	slots.AddPlayer(player)

	// Remove the player
	err := slots.RemovePlayer("player-1")
	if err != nil {
		t.Errorf("Expected no error removing player, got %v", err)
	}

	// Verify slot is now empty
	if slots.Slots[0].Player != nil {
		t.Error("Expected slot to be empty after removing player")
	}

	// Verify colour is preserved
	if slots.Slots[0].Colour != "blue" {
		t.Errorf("Expected slot colour to remain 'blue', got %s", slots.Slots[0].Colour)
	}

	// Try to remove non-existent player
	err = slots.RemovePlayer("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent player")
	}
}

// TestPlayerSlotsGetPlayers tests retrieving all players
func TestPlayerSlotsGetPlayers(t *testing.T) {
	slots := NewPlayerSlots()

	// Initially should return empty list
	players := slots.GetPlayers()
	if len(players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(players))
	}

	// Add two players
	player1 := &Player{ID: "player-1", Colour: "white"}
	player2 := &Player{ID: "player-2", Colour: "white"}

	slots.AddPlayer(player1)
	slots.AddPlayer(player2)

	// Should return 2 players
	players = slots.GetPlayers()
	if len(players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(players))
	}

	// Verify correct players are returned
	foundPlayer1 := false
	foundPlayer2 := false
	for _, p := range players {
		if p.ID == "player-1" {
			foundPlayer1 = true
		}
		if p.ID == "player-2" {
			foundPlayer2 = true
		}
	}

	if !foundPlayer1 || !foundPlayer2 {
		t.Error("Expected to find both player-1 and player-2 in results")
	}
}

// TestPlayerSlotsGetPlayer tests retrieving a specific player
func TestPlayerSlotsGetPlayer(t *testing.T) {
	slots := NewPlayerSlots()

	// Add players
	player1 := &Player{ID: "player-1", Colour: "white"}
	player2 := &Player{ID: "player-2", Colour: "white"}

	slots.AddPlayer(player1)
	slots.AddPlayer(player2)

	// Get existing player
	retrievedPlayer, found := slots.GetPlayer("player-1")
	if !found {
		t.Error("Expected to find player-1")
	}
	if retrievedPlayer.ID != "player-1" {
		t.Errorf("Expected player ID 'player-1', got %s", retrievedPlayer.ID)
	}

	// Try to get non-existent player
	_, found = slots.GetPlayer("non-existent")
	if found {
		t.Error("Expected not to find non-existent player")
	}
}

// TestPlayerSlotsColorAssignment tests that colours are preserved when adding/removing players
func TestPlayerSlotsColorAssignment(t *testing.T) {
	slots := NewPlayerSlots()

	// Add a player to first slot (should get blue)
	player1 := &Player{ID: "player-1", Colour: "white"}
	slots.AddPlayer(player1)

	// Verify slot colour is blue
	if slots.Slots[0].Colour != "blue" {
		t.Errorf("Expected first slot colour 'blue', got %s", slots.Slots[0].Colour)
	}

	// Remove player
	slots.RemovePlayer("player-1")

	// Add new player - should still go to first slot with blue colour
	player2 := &Player{ID: "player-2", Colour: "white"}
	slots.AddPlayer(player2)

	if slots.Slots[0].Colour != "blue" {
		t.Errorf("Expected first slot colour to remain 'blue', got %s", slots.Slots[0].Colour)
	}
	if slots.Slots[0].Player.ID != "player-2" {
		t.Errorf("Expected player-2 to be in first slot, got %s", slots.Slots[0].Player.ID)
	}
}
