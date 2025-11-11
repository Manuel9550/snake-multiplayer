package internal

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type State int

type Queue struct {
	data *list.List
}

func NewQueue() Queue {
	return Queue{
		data: list.New(),
	}
}

func (q *Queue) Enqueue(item Location) {
	if q.data == nil {
		q.data = list.New()
	}
	q.data.PushBack(item)
}

func (q *Queue) IsEmpty() bool {
	if q.data == nil {
		return true
	}
	return q.data.Len() == 0
}

func (q *Queue) Len() int {
	if q.data == nil {
		return 0
	}
	return q.data.Len()
}

func (q *Queue) Get(index int) *Location {
	if q.data == nil || index < 0 || index >= q.data.Len() {
		return nil
	}
	i := 0
	for e := q.data.Front(); e != nil; e = e.Next() {
		if i == index {
			loc := e.Value.(Location)
			return &loc
		}
		i++
	}
	return nil
}

func (q *Queue) Dequeue() *Location {
	if q.IsEmpty() {
		return nil
	}
	front := q.data.Front()
	q.data.Remove(front)
	loc := front.Value.(Location)
	return &loc
}

func (q *Queue) ToProto() []*models.BodyField {
	returnValues := []*models.BodyField{}
	if q != nil && q.data != nil {
		for e := q.data.Front(); e != nil; e = e.Next() {
			bodyField := e.Value.(Location)
			returnValues = append(returnValues, &models.BodyField{X: int32(bodyField.X), Y: int32(bodyField.Y)})
		}
	}
	return returnValues
}

type Player struct {
	conn      *websocket.Conn
	ID        string
	Colour    string
	Direction models.Direction
	X         int
	Y         int
	State     models.PlayerState
	BodyQueue Queue
	Length    int
	connMutex sync.Mutex
}

func NewPlayer(newConnection *websocket.Conn) *Player {
	bodyQueue := Queue{}
	bodyQueue.Enqueue(Location{X: 0, Y: 0})
	return &Player{
		ID:        uuid.New().String(),
		conn:      newConnection,
		Colour:    "white",
		Direction: models.Direction_DIRECTION_UP,
		X:         0,
		Y:         0,
		BodyQueue: bodyQueue,
		Length:    1,
		State:     models.PlayerState_STATE_CONNECTING,
		connMutex: sync.Mutex{},
	}
}

func (p *Player) ReadPlayerMessages(inputChannel chan []byte) error {
	if p.conn == nil {
		return nil
	}
	for {
		messageType, m, err := p.conn.ReadMessage()
		if err != nil {
			return err
		}

		switch messageType {
		case websocket.TextMessage:
			fmt.Println(string(m))
		case websocket.BinaryMessage:
			select {
			case inputChannel <- m:
				// Intentionally left blank
			default:
				fmt.Println("no message sent")
			}
		}
	}
}

func (p *Player) ShutDownPlayer() error {
	if p.conn == nil {
		return nil
	}
	return p.conn.Close()
}

func (p *Player) ToPlayerProtoMessage() *models.Player {
	return &models.Player{
		PlayerID: p.ID,
		Head: &models.BodyField{
			X: int32(p.X),
			Y: int32(p.Y),
		},
		Body:            p.BodyQueue.ToProto(),
		PlayerDirection: p.Direction,
		Colour:          p.Colour,
		State:           p.State,
	}
}

func (p *Player) UpdateDirection(newDirection models.Direction) {
	// Only change the player direction if it makes sense
	// I.e. snake can't start going left if it's current direction is right
	switch p.Direction {
	case models.Direction_DIRECTION_DOWN:
		if newDirection == models.Direction_DIRECTION_UP {
			return
		}
	case models.Direction_DIRECTION_UP:
		if newDirection == models.Direction_DIRECTION_DOWN {
			return
		}
	case models.Direction_DIRECTION_LEFT:
		if newDirection == models.Direction_DIRECTION_RIGHT {
			return
		}
	case models.Direction_DIRECTION_RIGHT:
		if newDirection == models.Direction_DIRECTION_LEFT {
			return
		}
	}

	p.Direction = newDirection
}

func (p *Player) UpdateState(newState models.PlayerState) {
	p.State = newState
}

func (p *Player) SendMessage(data []byte) {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()
	if p.conn == nil {
		return
	}
	if err := p.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		fmt.Printf("Error writing message for player ID %s:%s\n", p.ID, err.Error())
	}
}
