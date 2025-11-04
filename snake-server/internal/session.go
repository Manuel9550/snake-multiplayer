package internal

import (
	"fmt"
	"sync"
	"time"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const UpdateObjectsTime time.Duration = 1000 * time.Millisecond
const maxPlayers = 4

type Session struct {
	SessionID            string
	Entities             map[string]*Entity
	Players              map[string]*Player
	PlayerInputChannel   chan []byte
	playerMu             sync.Mutex
	Board                *Board
	UpdateEntitiesTicker *time.Ticker
	UpdateClientsTicker  *time.Ticker
	State                models.SessionState
	finishedChannel      chan string
}

func NewSession(finishedChannel chan string) *Session {
	session := Session{
		SessionID:            uuid.New().String(),
		Entities:             make(map[string]*Entity, 0),
		Players:              make(map[string]*Player, 0),
		playerMu:             sync.Mutex{},
		Board:                &Board{},
		PlayerInputChannel:   make(chan []byte),
		UpdateEntitiesTicker: time.NewTicker(UpdateObjectsTime),
		State:                models.SessionState_SESSION_STATE_INITIALIZING,
		finishedChannel:      finishedChannel,
	}
	fmt.Printf("New session %s\n", session.SessionID)

	go session.ReadChannelMessages()
	return &session
}

func (s *Session) AddPlayer(newPlayer *Player) error {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	_, ok := s.Players[newPlayer.ID]
	if ok {
		return fmt.Errorf("Player %s already exists in session", newPlayer.ID)
	}

	numberPlayer := len(s.Players) + 1
	switch numberPlayer {
	case 1:
		newPlayer.Colour = "blue"
	case 2:
		newPlayer.Colour = "green"
	case 3:
		newPlayer.Colour = "red"
	case 4:
		newPlayer.Colour = "yellow"
	default:
		newPlayer.Colour = "white"
	}

	newPlayerData := newPlayer.ToPlayerProtoMessage()
	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_PLAYER,
		MessageData: &models.Envelope_Data{
			Data: newPlayerData,
		},
	}
	data, err := proto.Marshal(&envelope)
	if err != nil {
		return err
	}

	if err := newPlayer.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		fmt.Println("Error writing message:", err)
	}

	newPlayer.State = models.State_STATE_CONNECTED
	s.Players[newPlayer.ID] = newPlayer
	fmt.Printf("Player ID %s added to session: %s\n", newPlayer.ID, s.SessionID)
	go s.HandlePlayer(newPlayer)
	return nil
}

func (s *Session) ReleasePlayer(playerID string) error {
	player, ok := s.Players[playerID]
	if !ok {
		return fmt.Errorf("Player %s no longer exists in session", playerID)
	}

	player.ShutDownPlayer()
	delete(s.Players, playerID)
	copiedMap := make(map[string]*Player, len(s.Players)-1)
	for key, value := range s.Players {
		if key != playerID {
			copiedMap[key] = value
		}
	}
	s.Players = copiedMap
	fmt.Printf("Player ID %s deleted from session: %s\n", player.ID, s.SessionID)
	if len(s.Players) == 0 {
		s.State = models.SessionState_SESSION_STATE_DONE
		select {
		case s.finishedChannel <- s.SessionID:
			fmt.Println("sent message", s.SessionID)
		default:
			fmt.Println("no message sent")
		}

	}

	return nil
}

func (s *Session) CanJoin() bool {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()

	return (len(s.Players) < maxPlayers || s.State == models.SessionState_SESSION_STATE_SETUP)
}

func (s *Session) StartSession() {
	s.playerMu.Lock()
	s.Board.Initialize(len(s.Players))
	if len(s.Players) == 0 {
		return
	}
	s.playerMu.Unlock()

	s.State = models.SessionState_SESSION_STATE_RUNNING

	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_SERVER_STATE,
		MessageData: &models.Envelope_ServerState{
			ServerState: &models.ServerStateMessage{
				SessionID:   s.SessionID,
				ServerState: models.ServerState(s.State),
			},
		},
	}

	data, err := proto.Marshal(&envelope)
	if err != nil {
		fmt.Println("Error marshalling message:", err)
	}

	s.BroadcastMessage(data)

	go s.UpdateState()
}

func (s *Session) HandlePlayer(player *Player) {
	// Send the player their welcome message
	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_WELCOME,
		MessageData: &models.Envelope_PlayerData{
			PlayerData: &models.Welcome{
				PlayerID:  player.ID,
				SessionID: s.SessionID,
				Colour:    player.Colour,
			},
		},
	}
	data, err := proto.Marshal(&envelope)
	if err != nil {
		fmt.Println("Error marshalling message:", err)
	}
	s.SendMessage(player.ID, data)

	// Broadcast that the player connected to all players
	s.BroadcastPlayerStates()

	// Read the player messages, until the connection is closed
	err = player.ReadPlayerMessages(s.PlayerInputChannel)
	// Now that we returned, was it a close error?
	if !websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
		// Log the error
		fmt.Printf("Player ID %s closed unexpectadly\n", player.ID)
	} else {
		fmt.Printf("Player ID %s closed connection\n", player.ID)
	}

	// Either way, shut down the player
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	s.ReleasePlayer(player.ID)
}

func (s *Session) ReadChannelMessages() {
	for message := range s.PlayerInputChannel {
		if err := s.HandleMessage(message); err != nil {
			fmt.Printf("Error reading message: %s : %s", string(message), err.Error())
		}
	}
}

func (s *Session) HandleMessage(message []byte) error {
	fmt.Printf("Got message: current state is: %s\n", s.State)
	// Depending on what state the session is in, we accept different types of messages
	switch s.State {
	case models.SessionState_SESSION_STATE_SETUP:
		err := s.HandlePlayerReadyUp(message)
		if err != nil {
			fmt.Println("Error handling player input message:", err)
		}
	case models.SessionState_SESSION_STATE_RUNNING:
		err := s.HandlePlayerInput(message)
		if err != nil {
			fmt.Println("Error handling player input message:", err)
		}
	}

	return nil
}

func (s *Session) HandlePlayerInput(message []byte) error {
	// This should be player input
	input := &models.Input{}
	if err := proto.Unmarshal(message, input); err != nil {
		return err
	}

	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	player, ok := s.Players[input.PlayerID]
	if !ok {
		return fmt.Errorf("Player ID %s doesn't exist in session", input.PlayerID)
	}

	player.UpdateDirection(input.InputType)

	return nil
}

func (s *Session) HandlePlayerReadyUp(message []byte) error {
	// This should be 'ready' message
	fmt.Printf("In HandlePlayerReadyUp\n")
	input := &models.PlayerState{}
	if err := proto.Unmarshal(message, input); err != nil {
		return err
	}

	s.playerMu.Lock()

	player, ok := s.Players[input.PlayerID]
	if !ok {
		return fmt.Errorf("Player ID %s doesn't exist in session", input.PlayerID)
	}

	if input.State != models.State_STATE_READY {
		// Intentionally ignore messages that aren't readying up
		fmt.Printf("Ignoring non ready up message, got: %v", input.State)
		return nil
	}

	player.UpdateState(input.State)
	s.playerMu.Unlock()

	s.BroadcastPlayerStates()

	if s.CheckAllPlayersReady() {
		s.StartSession()
	}

	return nil
}

func (s *Session) CheckAllPlayersReady() bool {
	for _, player := range s.Players {
		if player.State != models.State_STATE_READY {
			return false
		}
	}
	return true
}

func (s *Session) UpdateState() {
	for range s.UpdateEntitiesTicker.C {
		s.UpdateBoard()
		s.UpdateClients()
	}
}

func (s *Session) UpdateBoard() {
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	for _, player := range s.Players {
		if player.State != models.State_STATE_DONE {
			s.Board.Update(player)
		}
	}
}

func (s *Session) UpdateClients() {
	s.playerMu.Lock()
	gameData := models.GameData{
		PlayerData: []*models.Player{},
		EntityData: []*models.Entity{},
	}
	for _, player := range s.Players {
		gameData.PlayerData = append(gameData.PlayerData, player.ToPlayerProtoMessage())
	}
	s.playerMu.Unlock()
	for location, entity := range s.Board.Entities {
		gameData.EntityData = append(gameData.EntityData, entity.ToEntityProtoMessage(location))
	}
	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_GAME_DATA,
		MessageData: &models.Envelope_GameState{
			GameState: &gameData,
		},
	}
	data, err := proto.Marshal(&envelope)
	if err != nil {
		fmt.Println("Error marshalling message:", err)
	}

	s.BroadcastMessage(data)

}

func (s *Session) SendMessage(playerID string, data []byte) {
	s.playerMu.Lock()
	player, ok := s.Players[playerID]
	if !ok || player == nil {
		fmt.Printf("Error finding player with ID %s\n", playerID)
	}

	if err := player.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		fmt.Printf("Error writing message for player ID %s:%s\n", player.ID, err.Error())
	}

	s.playerMu.Unlock()
}

func (s *Session) BroadcastMessage(data []byte) {
	s.playerMu.Lock()
	for _, player := range s.Players {
		if err := player.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			fmt.Printf("Error writing message for player ID %s:%s\n", player.ID, err.Error())
		}
	}
	s.playerMu.Unlock()
}

func (s *Session) DeleteSessionResources() {
	// Release all players, if they haven't been released already
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	for _, player := range s.Players {
		err := s.ReleasePlayer(player.ID)
		if err != nil {
			fmt.Printf("Error releasing player ID %s:%s", player.ID, err.Error())
		}
	}

	close(s.PlayerInputChannel)
}

func (s *Session) PlayerStates() models.PlayerStates {
	returnValues := models.PlayerStates{
		PlayerStates: make([]*models.PlayerState, 0),
	}
	s.playerMu.Lock()
	defer s.playerMu.Unlock()
	for _, player := range s.Players {
		returnValues.PlayerStates = append(returnValues.PlayerStates, &models.PlayerState{
			PlayerID: player.ID,
			State:    player.State,
			Colour:   player.Colour,
		})
	}

	return returnValues
}

func (s *Session) BroadcastPlayerStates() {
	states := s.PlayerStates()
	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_PLAYER_STATES,
		MessageData: &models.Envelope_PlayerStates{
			PlayerStates: &states,
		},
	}
	data, err := proto.Marshal(&envelope)
	if err != nil {
		fmt.Println("Error marshalling player states message:", err)
	}
	s.BroadcastMessage(data)
}
