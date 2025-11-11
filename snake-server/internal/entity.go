package internal

import (
	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/google/uuid"
)

const NeutralID = "NEUTRAL_PLAYER"

type Location struct {
	X int
	Y int
}

func (e *Location) ToBodyFieldProtoMessage() *models.BodyField {
	return &models.BodyField{
		X: int32(e.X),
		Y: int32(e.Y),
	}
}

type Entity struct {
	ID       string
	PlayerID string
	Type     models.EntityType
	Location Location
}

func NewEntity(playerID string, entityType models.EntityType, location Location) Entity {
	return Entity{
		ID:       uuid.New().String(),
		PlayerID: playerID,
		Type:     entityType,
		Location: location,
	}
}

func (e *Entity) ToEntityProtoMessage() *models.Entity {
	return &models.Entity{
		ID:       e.ID,
		PlayerID: e.PlayerID,
		Body:     e.Location.ToBodyFieldProtoMessage(),
		Type:     e.Type,
	}
}
