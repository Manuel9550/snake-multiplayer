package internal

import (
	"math/rand"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
)

const numRows = 20
const numColumns = 20

type Board struct {
	Tiles    [][]*Entity
	Entities map[EntityLocation]Entity
}

func (b *Board) Initialize(players []*Player) {
	b.Tiles = make([][]*Entity, numColumns)
	for i := range b.Tiles {
		b.Tiles[i] = make([]*Entity, numRows)
	}

	b.Entities = make(map[EntityLocation]Entity)

	for playerNumber, player := range players {
		switch playerNumber {
		case 0:
			player.X = 1
			player.Y = 1
			player.Direction = models.Direction_DIRECTION_RIGHT
		case 1:
			player.X = numColumns - 2
			player.Y = 1
			player.Direction = models.Direction_DIRECTION_DOWN
		case 2:
			player.X = numColumns - 2
			player.Y = numRows - 2
			player.Direction = models.Direction_DIRECTION_LEFT
		case 3:
			player.X = 1
			player.Y = numRows - 2
			player.Direction = models.Direction_DIRECTION_UP
		}
		player.BodyQueue.Enqueue(EntityLocation{X: player.X, Y: player.Y})
		b.Entities[EntityLocation{X: player.X, Y: player.Y}] = NewEntity(player.ID, models.EntityType_ENTITY_TYPE_BODY)
	}

	for i := 0; i < len(players)+1; i++ {
		b.AddNewPoint()
	}
	for i := 0; i < 10; i++ {
		b.AddNewWall()
	}
}

func (b *Board) Update(player *Player) {
	if player.State == models.State_STATE_DONE {
		return
	}

	newLocation := b.NewLocation(player)
	player.X = newLocation.X
	player.Y = newLocation.Y

	// Did the player hit something?
	hitEntity, ok := b.Entities[newLocation]
	if ok {
		// The player hit something, was it a point?
		if hitEntity.Type == models.EntityType_ENTITY_TYPE_POINT {
			player.Length += 1
			delete(b.Entities, newLocation)
			b.AddNewPoint()
		} else {
			b.PlayerFinished(player)
		}
	}

	if player.State != models.State_STATE_DONE {
		player.BodyQueue.Enqueue(EntityLocation{X: player.X, Y: player.Y})
		b.Entities[newLocation] = NewEntity(player.ID, models.EntityType_ENTITY_TYPE_BODY)

		if len(player.BodyQueue) > player.Length {
			oldPiece := player.BodyQueue.Dequeue()
			delete(b.Entities, *oldPiece)
		}
	}
}

func (b *Board) PlayerFinished(player *Player) {
	player.State = models.State_STATE_DONE
	for !player.BodyQueue.IsEmpty() {
		oldPiece := player.BodyQueue.Dequeue()
		delete(b.Entities, *oldPiece)
	}
}

// Remember, on the front end, UP is -Y, and DOWN is +Y
func (b *Board) NewLocation(player *Player) EntityLocation {
	locationX, locationY := player.X, player.Y
	switch player.Direction {
	case models.Direction_DIRECTION_DOWN:
		locationY += 1
	case models.Direction_DIRECTION_UP:
		locationY -= 1
	case models.Direction_DIRECTION_LEFT:
		locationX -= 1
	case models.Direction_DIRECTION_RIGHT:
		locationX += 1
	}

	if locationY >= numRows {
		locationY = 0
	}
	if locationY < 0 {
		locationY = numRows - 1
	}

	if locationX >= numColumns {
		locationX = 0
	}
	if locationX < 0 {
		locationX = numColumns - 1
	}

	return EntityLocation{X: locationX, Y: locationY}
}

func (b *Board) AddNewPoint() {
	location := b.GetRandomFreeSpace()
	b.Entities[location] = NewEntity(NeutralID, models.EntityType_ENTITY_TYPE_POINT)
}

func (b *Board) AddNewWall() {
	location := b.GetRandomFreeSpace()
	b.Entities[location] = NewEntity(NeutralID, models.EntityType_ENTITY_TYPE_WALL)
}

func (b *Board) GetRandomFreeSpace() EntityLocation {
	for {
		randomX := rand.Intn(numColumns)
		randomY := rand.Intn(numRows)
		newLocation := EntityLocation{X: randomX, Y: randomY}
		_, ok := b.Entities[newLocation]
		if !ok {
			return newLocation
		}
	}
}
