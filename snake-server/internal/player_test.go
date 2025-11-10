package internal

import (
	"testing"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
)

// TestQueueEnqueue tests adding items to the queue
func TestQueueEnqueue(t *testing.T) {
	q := Queue{}

	q.Enqueue(EntityLocation{X: 1, Y: 2})
	q.Enqueue(EntityLocation{X: 3, Y: 4})

	if len(q) != 2 {
		t.Errorf("Expected queue length 2, got %d", len(q))
	}

	if q[0].X != 1 || q[0].Y != 2 {
		t.Errorf("Expected first element {1, 2}, got {%d, %d}", q[0].X, q[0].Y)
	}

	if q[1].X != 3 || q[1].Y != 4 {
		t.Errorf("Expected second element {3, 4}, got {%d, %d}", q[1].X, q[1].Y)
	}
}

// TestQueueDequeue tests removing items from the queue
func TestQueueDequeue(t *testing.T) {
	q := Queue{}
	q.Enqueue(EntityLocation{X: 1, Y: 2})
	q.Enqueue(EntityLocation{X: 3, Y: 4})

	item := q.Dequeue()
	if item == nil {
		t.Fatal("Expected non-nil item from Dequeue")
	}

	if item.X != 1 || item.Y != 2 {
		t.Errorf("Expected dequeued element {1, 2}, got {%d, %d}", item.X, item.Y)
	}

	if len(q) != 1 {
		t.Errorf("Expected queue length 1 after dequeue, got %d", len(q))
	}

	item = q.Dequeue()
	if item == nil {
		t.Fatal("Expected non-nil item from second Dequeue")
	}

	if item.X != 3 || item.Y != 4 {
		t.Errorf("Expected dequeued element {3, 4}, got {%d, %d}", item.X, item.Y)
	}

	if len(q) != 0 {
		t.Errorf("Expected empty queue after two dequeues, got length %d", len(q))
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

	q.Enqueue(EntityLocation{X: 1, Y: 2})

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
	q.Enqueue(EntityLocation{X: 5, Y: 10})
	q.Enqueue(EntityLocation{X: 15, Y: 20})

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
		State: models.State_STATE_CONNECTING,
	}

	player.UpdateState(models.State_STATE_CONNECTED)
	if player.State != models.State_STATE_CONNECTED {
		t.Errorf("Expected state STATE_CONNECTED, got %v", player.State)
	}

	player.UpdateState(models.State_STATE_READY)
	if player.State != models.State_STATE_READY {
		t.Errorf("Expected state STATE_READY, got %v", player.State)
	}

	player.UpdateState(models.State_STATE_DONE)
	if player.State != models.State_STATE_DONE {
		t.Errorf("Expected state STATE_DONE, got %v", player.State)
	}
}

// TestToPlayerProtoMessage tests converting player to proto message
func TestToPlayerProtoMessage(t *testing.T) {
	bodyQueue := Queue{}
	bodyQueue.Enqueue(EntityLocation{X: 5, Y: 10})
	bodyQueue.Enqueue(EntityLocation{X: 6, Y: 10})

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
