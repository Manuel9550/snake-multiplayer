package internal

import (
	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/google/uuid"
)

type EntityType int

const NeutralID = "NEUTRAL_PLAYER"

type EntityLocation struct {
	X int
	Y int
}

func (e *EntityLocation) ToBodyFieldProtoMessage() *models.BodyField {
	return &models.BodyField{
		X: int32(e.X),
		Y: int32(e.Y),
	}
}

type Entity struct {
	ID       string
	PlayerID string
	Type     models.EntityType
}

func NewEntity(playerID string, entityType models.EntityType) Entity {
	return Entity{
		ID:       uuid.New().String(),
		PlayerID: playerID,
		Type:     entityType,
	}
}

func (e *Entity) ToEntityProtoMessage(location EntityLocation) *models.Entity {
	return &models.Entity{
		ID:       e.ID,
		PlayerID: e.PlayerID,
		Body:     location.ToBodyFieldProtoMessage(),
		Type:     e.Type,
	}
}
