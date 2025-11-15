package internal

import (
	"fmt"
	"math/rand"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
)

const numRows = 20
const numColumns = 20
const minWalls = 10
const maxWalls = 40

// Percentage chance the players will use a pre-generated layout
const preLayoutPercentage = 70

type Board struct {
	Tiles    [][]*Entity
	Entities map[Location]Entity
}

var startingLocations = []Location{
	// Player 1
	{X: 2, Y: 2},
	// Player 2
	{X: numColumns - 3, Y: 2},
	// Player 3
	{X: numColumns - 3, Y: numRows - 3},
	// Player 4
	{X: 2, Y: numRows - 3},
}

var allowedWallSpaces = initializeAllowedWallSpaces()

func initializeAllowedWallSpaces() []Location {
	allowedSpaces := make([]Location, 0)
	for x := 0; x < numColumns; x++ {
		for y := 0; y < numColumns; y++ {
			// check for player starting locations

			// Player 1 points to the right
			if y == startingLocations[0].Y && x >= startingLocations[0].X-1 && x <= startingLocations[0].X+3 {
				continue
			}

			// Player 2 points downwards
			if x == startingLocations[1].X && y >= startingLocations[1].Y-1 && y <= startingLocations[1].Y+3 {
				continue
			}

			// Player 3 points to the left
			if y == startingLocations[2].Y && x >= startingLocations[2].X-3 && x <= startingLocations[2].X+1 {
				continue
			}

			// Player 4 points upwards
			if x == startingLocations[3].X && y >= startingLocations[3].Y+1 && y <= startingLocations[3].Y-3 {
				continue
			}

			allowedSpaces = append(allowedSpaces, Location{X: x, Y: y})
		}
	}

	return allowedSpaces
}

func (b *Board) Initialize(playerSlots PlayerSlots) {
	b.Entities = make(map[Location]Entity)

	for playerNumber, playerSlot := range playerSlots.Slots {
		// Only want to initialize if a player exists in this slot
		if playerSlot.Player == nil {
			continue
		}

		player := playerSlot.Player
		var trailingLocation = Location{}
		player.X = startingLocations[playerNumber].X
		player.Y = startingLocations[playerNumber].Y
		switch playerNumber {
		case 0:
			player.Direction = models.Direction_DIRECTION_RIGHT
			trailingLocation.X = player.X - 1
			trailingLocation.Y = player.Y
		case 1:
			player.Direction = models.Direction_DIRECTION_DOWN
			trailingLocation.X = player.X
			trailingLocation.Y = player.Y - 1
		case 2:
			player.Direction = models.Direction_DIRECTION_LEFT
			trailingLocation.X = player.X + 1
			trailingLocation.Y = player.Y
		case 3:
			player.Direction = models.Direction_DIRECTION_UP
			trailingLocation.X = player.X
			trailingLocation.Y = player.Y + 1
		}
		location := Location{X: player.X, Y: player.Y}
		player.BodyQueue.Enqueue(trailingLocation)
		player.BodyQueue.Enqueue(location)
		b.Entities[location] = NewEntity(player.ID, models.EntityType_ENTITY_TYPE_BODY, location)
		b.Entities[trailingLocation] = NewEntity(player.ID, models.EntityType_ENTITY_TYPE_BODY, trailingLocation)
		player.State = models.PlayerState_STATE_PLAYING
		player.Length = 2
	}

	b.CreateLayout()
	for i := 0; i < len(playerSlots.GetPlayers())+1; i++ {
		b.AddNewPoint()
	}
}

func (b *Board) CreateLayout() {
	randomLayout := rand.Intn(100)
	if len(PreMadeLayouts) == 0 || randomLayout >= preLayoutPercentage {
		// Just create random walls for the players
		numWalls := rand.Intn(maxWalls-minWalls+1) + minWalls
		for i := 0; i <= numWalls; i++ {
			b.AddNewWall()
		}

		return
	}

	// Use a premade layout
	layoutToUseIndex := rand.Intn(len(PreMadeLayouts))
	layoutToUse := PreMadeLayouts[layoutToUseIndex]
	fmt.Printf("using layout: %s", layoutToUse.Name)
	for _, location := range layoutToUse.Walls {
		b.Entities[location] = NewEntity(NeutralID, models.EntityType_ENTITY_TYPE_WALL, location)
	}
}

func (b *Board) GenerateRandomLayout() {
	fmt.Printf("In gnerate random layout")
	numWalls := rand.Intn(maxWalls-minWalls+1) + minWalls
	fmt.Printf("NumWalls : %d", numWalls)
	for i := 0; i <= numWalls; i++ {
		b.AddNewWall()
	}
}

func (b *Board) Update(player *Player) {
	if player.State == models.PlayerState_STATE_DONE {
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

	if player.State != models.PlayerState_STATE_DONE {
		location := Location{X: player.X, Y: player.Y}
		player.BodyQueue.Enqueue(location)
		b.Entities[newLocation] = NewEntity(player.ID, models.EntityType_ENTITY_TYPE_BODY, location)

		if player.BodyQueue.Len() > player.Length {
			oldPiece := player.BodyQueue.Dequeue()
			delete(b.Entities, *oldPiece)
		}
	}
}

func (b *Board) PlayerFinished(player *Player) {
	player.State = models.PlayerState_STATE_DONE
	for !player.BodyQueue.IsEmpty() {
		oldPiece := player.BodyQueue.Dequeue()
		delete(b.Entities, *oldPiece)
	}
}

// Remember, on the front end, UP is -Y, and DOWN is +Y
func (b *Board) NewLocation(player *Player) Location {
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

	return Location{X: locationX, Y: locationY}
}

func (b *Board) AddNewPoint() {
	location := b.GetRandomFreeSpace()
	b.Entities[location] = NewEntity(NeutralID, models.EntityType_ENTITY_TYPE_POINT, location)
}

func (b *Board) AddNewWall() {
	for {
		randomWallIndex := rand.Intn(len(allowedWallSpaces))
		randomWallLocation := allowedWallSpaces[randomWallIndex]
		_, ok := b.Entities[randomWallLocation]
		if !ok {
			b.Entities[randomWallLocation] = NewEntity(NeutralID, models.EntityType_ENTITY_TYPE_WALL, randomWallLocation)
			return
		}
	}
}

func (b *Board) GetRandomFreeSpace() Location {
	for {
		randomX := rand.Intn(numColumns)
		randomY := rand.Intn(numRows)
		newLocation := Location{X: randomX, Y: randomY}
		_, ok := b.Entities[newLocation]
		if !ok {
			return newLocation
		}
	}
}

func (b *Board) GetEntities() []*Entity {
	returnEntities := make([]*Entity, 0)
	for _, entity := range b.Entities {
		returnEntities = append(returnEntities, &entity)
	}
	return returnEntities
}
