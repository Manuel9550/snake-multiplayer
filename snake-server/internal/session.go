package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const UpdateObjectsTime = 1000 * time.Millisecond
const maxPlayers = 4

// ErrSessionShuttingDown is returned by the public actor methods once the session's ctx has been cancelled.
var ErrSessionShuttingDown = errors.New("session is shutting down")

type Session struct {
	SessionID            string
	PlayerSlots          PlayerSlots
	PlayerInputChannel   chan []byte
	Board                *Board
	UpdateEntitiesTicker *time.Ticker
	UpdateClientsTicker  *time.Ticker
	State                models.SessionState
	finishedChannel      chan string
	winningPlayerID      string

	// Single actor pattern channel
	// Public methods will make a closure that the actor (session) will do
	sessionInputCh chan func()
	shutdownCh     chan struct{}

	// To handle shutting down the session
	ctx    context.Context
	cancel context.CancelFunc

	// actorWg tracks run() and ReadChannelMessages()
	// These are controlled by the session directly
	actorWg sync.WaitGroup

	// connWg tracks HandlePlayer (one per player) and UpdateState. DeleteSession waits for these
	// only after closing every remaining connection
	connWg sync.WaitGroup

	logger *log.Logger
}

func NewSession(finishedChannel chan string, logger *log.Logger) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	session := Session{
		SessionID:            uuid.New().String(),
		Board:                &Board{},
		PlayerInputChannel:   make(chan []byte),
		UpdateEntitiesTicker: time.NewTicker(UpdateObjectsTime),
		State:                models.SessionState_SESSION_STATE_INITIALIZING,
		finishedChannel:      finishedChannel,
		PlayerSlots:          NewPlayerSlots(),

		// Initialize the actual session input channel
		// Buffered so broadcasts/sends queued up in quick succession don't block their callers waiting for the actor to catch up
		sessionInputCh: make(chan func(), 100),
		shutdownCh:     make(chan struct{}, 1), // Buffered to allow non-blocking sends

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
	}
	logger.Printf("New session %s\n", session.SessionID)

	session.actorWg.Add(2)
	go func() {
		// The main actor goroutine
		defer session.actorWg.Done()
		session.run()
	}()
	go func() {
		// Reading messages players send
		defer session.actorWg.Done()
		session.ReadChannelMessages()
	}()

	return &session
}

// run is the central goroutine that owns the Players map
func (s *Session) run() {
	for {
		select {
		// The session state needs to be changed
		case fn := <-s.sessionInputCh:
			fn()

		case <-s.shutdownCh:
			// Whether the game was finished, or all players left, this session is over
			// Tell the session manager to delete this session
			// The actual session resources will clean up after this goroutine is done
			s.initiateShutDown()
			s.finishedChannel <- s.SessionID
			return
		}
	}
}

// Internal methods - called only by run() goroutine

func (s *Session) addPlayerInternal(newPlayer *Player) error {

	if _, ok := s.PlayerSlots.GetPlayer(newPlayer.ID); ok {
		return fmt.Errorf("player %s already exists in session", newPlayer.ID)
	}

	success := s.PlayerSlots.AddPlayer(newPlayer)
	if !success {
		return fmt.Errorf("could not add player %s to session %s", newPlayer.ID, s.SessionID)
	}
	s.logger.Printf("Player ID %s added to session: %s\n", newPlayer.ID, s.SessionID)

	s.connWg.Add(1)
	go s.HandlePlayer(newPlayer)
	return nil
}

func (s *Session) removePlayerInternal(playerID string) error {
	player, ok := s.PlayerSlots.GetPlayer(playerID)
	if !ok {
		return fmt.Errorf("player %s no longer exists in session %s when trying to remove", playerID, s.SessionID)
	}

	err := s.PlayerSlots.RemovePlayer(playerID)
	if err != nil {
		s.logger.Printf("error when attempting to shut down player %s: %v", playerID, err)
	}
	s.logger.Printf("Player ID %s deleted from session: %s\n", player.ID, s.SessionID)

	return nil
}

func (s *Session) broadcastInternal(data []byte) {
	// Create snapshot of players to avoid holding up the actor while doing I/O
	players := s.PlayerSlots.GetPlayers()

	// Do I/O outside the actor select loop (in a goroutine for non-blocking)
	go func() {
		for _, player := range players {
			player.SendMessage(data)
		}
	}()
}

// broadcastAndWait sends data to players synchronously and blocks until delivery is done, unlike BroadcastMessage. Need it for the final message that must land before the session shuts down.
func (s *Session) broadcastAndWait(players []*Player, data []byte) {
	doneCh := make(chan struct{}, 1)
	if !s.sendToActor(func() {
		for _, player := range players {
			player.SendMessage(data)
		}
		doneCh <- struct{}{}
	}) {
		return
	}
	awaitResult(s, doneCh, struct{}{})
}

func (s *Session) sendMessageInternal(playerID *string, data []byte) {
	// If playerID is nil, we just broadcast to everyone
	if playerID == nil {
		s.broadcastInternal(data)
		return
	}

	player, ok := s.PlayerSlots.GetPlayer(*playerID)
	if !ok || player == nil {
		s.logger.Printf("Error finding player with ID %s\n", *playerID)
		return
	}

	// Do I/O in goroutine to avoid blocking the actor
	go func() {
		player.SendMessage(data)
	}()
}

// Public methods will communicate with the session by calling the sessionInputCh

// sendToActor hands fn to the actor goroutine, returning false instead of blocking forever
// if the session has already begun shutting down.
func (s *Session) sendToActor(fn func()) bool {
	select {
	case s.sessionInputCh <- fn:
		return true
	case <-s.ctx.Done():
		return false
	}
}

// awaitResult waits for resultCh, but gives up and returns fallback if the session shuts down
// before the actor gets around to running the closure that would have produced a result.
func awaitResult[T any](s *Session, resultCh <-chan T, fallback T) T {
	select {
	case v := <-resultCh:
		return v
	case <-s.ctx.Done():
		return fallback
	}
}

func (s *Session) shuttingDownErr() error {
	return fmt.Errorf("session %s: %w", s.SessionID, ErrSessionShuttingDown)
}

func (s *Session) AddPlayer(newPlayer *Player) error {
	resultCh := make(chan error, 1)
	if !s.sendToActor(func() {
		resultCh <- s.addPlayerInternal(newPlayer)
		// Every time a player is added/deleted, update the other players
		players := s.PlayerSlots.GetPlayers()
		entities := s.Board.GetEntities()
		s.broadcastInternal(s.GetSessionData(players, entities))
	}) {
		return s.shuttingDownErr()
	}
	return awaitResult(s, resultCh, s.shuttingDownErr())
}

func (s *Session) ReleasePlayer(playerID string) error {
	resultCh := make(chan error, 1)
	if !s.sendToActor(func() {
		resultCh <- s.removePlayerInternal(playerID)
		// We can end the session early if there are no more players left
		players := s.PlayerSlots.GetPlayers()
		if len(players) == 0 {
			s.setSessionDone(nil)
		}
		entities := s.Board.GetEntities()
		s.broadcastInternal(s.GetSessionData(players, entities))

		if s.State == models.SessionState_SESSION_STATE_DONE {
			s.shutdownCh <- struct{}{}
		}
	}) {
		return s.shuttingDownErr()
	}
	return awaitResult(s, resultCh, s.shuttingDownErr())
}

func (s *Session) CanJoin() bool {
	resultCh := make(chan bool, 1)
	if !s.sendToActor(func() {
		resultCh <- len(s.PlayerSlots.GetPlayers()) < maxPlayers && s.State == models.SessionState_SESSION_STATE_SETUP
	}) {
		return false
	}
	return awaitResult(s, resultCh, false)
}

func (s *Session) StartSession() {
	players := s.GetPlayers()

	if len(players) == 0 {
		return
	}

	s.Board.Initialize(s.PlayerSlots)
	s.State = models.SessionState_SESSION_STATE_RUNNING
	s.BroadcastMessage(s.GetSessionData(nil, nil))

	s.connWg.Add(1)
	go s.UpdateState()
}

func (s *Session) HandlePlayer(player *Player) {
	defer s.connWg.Done()

	// If conn is nil, this is a test scenario - don't auto-remove the player
	if player.conn == nil {
		// Still broadcast player states for testing purposes
		s.BroadcastMessage(s.GetSessionData(nil, nil))
		return
	}

	// Send the player their welcome message
	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_WELCOME,
		MessageData: &models.Envelope_Welcome{
			Welcome: player.ToPlayerProtoMessage(),
		},
	}
	data, err := proto.Marshal(&envelope)
	if err != nil {
		s.logger.Printf("Error marshalling message: %s", err.Error())
	}
	s.SendMessage(&player.ID, data)

	// Read the player messages, until the connection is closed
	err = player.ReadPlayerMessages(s.PlayerInputChannel)
	switch {
	case websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure):
		s.logger.Printf("Player ID %s closed connection unexpectedly: %s\n", player.ID, err.Error())
	case s.ctx.Err() != nil:
		// Session already closed this connection itself during teardown
	default:
		s.logger.Printf("Player ID %s closed connection\n", player.ID)
	}

	err = s.ReleasePlayer(player.ID)
	if err != nil && !errors.Is(err, ErrSessionShuttingDown) {
		s.logger.Printf("Error releasing Player ID %s: %s\n", player.ID, err.Error())
	}

}

func (s *Session) ReadChannelMessages() {
	for {
		select {
		case msg := <-s.PlayerInputChannel:
			if err := s.HandleMessage(msg); err != nil {
				s.logger.Printf("Error reading message: %s : %s", string(msg), err.Error())
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Session) HandleMessage(message []byte) error {
	// Depending on what state the session is in, we accept different types of messages
	switch s.State {
	case models.SessionState_SESSION_STATE_SETUP:
		err := s.HandlePlayerReadyUp(message)
		if err != nil {
			s.logger.Printf("Error handling player input message: %s", err.Error())
		}
	case models.SessionState_SESSION_STATE_RUNNING:
		err := s.HandlePlayerInput(message)
		if err != nil {
			s.logger.Printf("Error handling player input message: %s", err.Error())
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

	return s.UpdatePlayerDirection(input.PlayerID, input.InputType)
}

func (s *Session) HandlePlayerReadyUp(message []byte) error {
	// This should be 'ready' message
	input := &models.ReadyUp{}
	if err := proto.Unmarshal(message, input); err != nil {
		return err
	}

	if err := s.UpdatePlayerState(input.PlayerID, models.PlayerState_STATE_READY); err != nil {
		return err
	}

	sessionData := s.GetSessionData(nil, nil)
	s.BroadcastMessage(sessionData)

	if s.CheckAllPlayersReady() {
		s.StartSession()
	}

	return nil
}

func (s *Session) UpdatePlayerDirection(playerID string, direction models.Direction) error {
	resultCh := make(chan error, 1)
	if !s.sendToActor(func() {
		player, ok := s.PlayerSlots.GetPlayer(playerID)
		if !ok {
			resultCh <- fmt.Errorf("player ID %s doesn't exist in session", playerID)
		} else {
			player.UpdateDirection(direction)
			resultCh <- nil
		}
	}) {
		return s.shuttingDownErr()
	}
	return awaitResult(s, resultCh, s.shuttingDownErr())
}

func (s *Session) UpdatePlayerState(playerID string, state models.PlayerState) error {
	resultCh := make(chan error, 1)
	if !s.sendToActor(func() {
		player, ok := s.PlayerSlots.GetPlayer(playerID)
		if !ok {
			resultCh <- fmt.Errorf("player ID %s doesn't exist in session", playerID)
		} else {
			player.UpdateState(state)
			resultCh <- nil
		}
	}) {
		return s.shuttingDownErr()
	}
	return awaitResult(s, resultCh, s.shuttingDownErr())
}

func (s *Session) CheckAllPlayersReady() bool {
	resultCh := make(chan bool, 1)
	if !s.sendToActor(func() {
		allReady := true
		playerCount := 0
		for _, player := range s.PlayerSlots.GetPlayers() {
			playerCount += 1
			if player.State != models.PlayerState_STATE_READY {
				allReady = false
				break
			}
		}
		if playerCount < 2 {
			allReady = false
		}

		resultCh <- allReady
	}) {
		return false
	}
	return awaitResult(s, resultCh, false)
}

func (s *Session) UpdateState() {
	defer s.connWg.Done()
	defer func() {
		s.shutdownCh <- struct{}{}
	}()
	for range s.UpdateEntitiesTicker.C {
		players := s.GetPlayers()

		// If all players have left, there is no point in continuing
		// mark session as done and trigger shutdown
		if len(players) == 0 {
			s.setSessionDone(nil)
			s.logger.Printf("All players left session %s, marking as DONE", s.SessionID)
			return
		}

		s.UpdateBoard(players)
		gameDone, winningPlayer := s.CheckGameOver(players)
		if gameDone {
			s.setSessionDone(winningPlayer)
			sessionData := s.GetSessionData(players, nil)
			s.broadcastAndWait(players, sessionData)
			return
		}
		sessionData := s.GetSessionData(players, nil)
		s.BroadcastMessage(sessionData)
	}
}

func (s *Session) setSessionDone(winningPlayer *Player) {
	s.State = models.SessionState_SESSION_STATE_DONE
	if winningPlayer != nil {
		s.winningPlayerID = winningPlayer.ID
	}
}

func (s *Session) UpdateBoard(players []*Player) {
	for _, player := range players {
		if player.State != models.PlayerState_STATE_DONE {
			s.Board.Update(player)
		}
	}
}

func (s *Session) CheckGameOver(players []*Player) (bool, *Player) {
	activePlayers := 0
	var activePlayer *Player
	for _, player := range players {
		if player.State != models.PlayerState_STATE_DONE {
			activePlayers += 1
			activePlayer = player
		}
	}

	if activePlayers == 1 {
		return true, activePlayer
	}
	if activePlayers == 0 {
		return true, nil
	}

	return false, nil
}

func (s *Session) SendMessage(playerID *string, data []byte) {
	s.sendToActor(func() {
		s.sendMessageInternal(playerID, data)
	})
}

func (s *Session) BroadcastMessage(data []byte) {
	// If there isn't a specific player ID, we just broadcast to everyone
	s.sendToActor(func() {
		s.sendMessageInternal(nil, data)
	})
}

func (s *Session) GetPlayers() []*Player {
	resultCh := make(chan []*Player, 1)
	if !s.sendToActor(func() {
		resultCh <- s.PlayerSlots.GetPlayers()
	}) {
		return nil
	}
	return awaitResult(s, resultCh, ([]*Player)(nil))
}

func (s *Session) GetEntities() []*Entity {
	resultCh := make(chan []*Entity, 1)
	if !s.sendToActor(func() {
		resultCh <- s.Board.GetEntities()
	}) {
		return nil
	}
	return awaitResult(s, resultCh, ([]*Entity)(nil))
}

// Called from main actor goroutine: prepare the session for deletion from the sessionManager
func (s *Session) initiateShutDown() {
	s.cancel() // SessionManager can read the context to know this session is cancelled (done)
	s.UpdateEntitiesTicker.Stop()
}

// Discard tears down a session that was never registered with a SessionManager
// Not a likely scenario,
func (s *Session) Discard() {
	// shutdownCh is buffered and this session never had a player, so nothing else could have signalled it yet
	s.shutdownCh <- struct{}{}
	s.DeleteSession()
}

// DeleteSession is called by the session manager, actually clean up the session
func (s *Session) DeleteSession() {
	s.logger.Printf("Shutting down session: %s", s.SessionID)
	// Wait for run() and ReadChannelMessages() to finish before touching PlayerSlots ourselves
	s.actorWg.Wait()

	for _, player := range s.PlayerSlots.GetPlayers() {
		// We are still in the main goroutine, release the players.
		// This closes each remaining connection, which is what unblocks any HandlePlayer
		// goroutine still parked in conn.ReadMessage().
		if err := s.removePlayerInternal(player.ID); err != nil {
			// even if it errors, need to continue
			s.logger.Printf("error removing player %s:%v", player.ID, err)
		}
	}

	// Now that every connection is closed, HandlePlayer/UpdateState are guaranteed to finish soon.
	// Wait for them before closing PlayerInputChannel so nothing can still be sending into it.
	s.connWg.Wait()

	close(s.PlayerInputChannel)
}

func (s *Session) GetSessionData(players []*Player, entities []*Entity) []byte {
	// Depending on where we are in the session, we may pass in the players and entities
	if players == nil {
		players = s.GetPlayers()
	}

	if entities == nil {
		entities = s.GetEntities()
	}

	returnData := models.Envelope_SessionData{
		SessionData: &models.SessionData{
			SessionID:     s.SessionID,
			State:         s.State,
			WinningPlayer: s.winningPlayerID,
			GameData: &models.GameData{
				PlayerData: make([]*models.Player, 0),
				EntityData: make([]*models.Entity, 0),
			},
		},
	}

	for _, player := range players {
		returnData.SessionData.GameData.PlayerData = append(returnData.SessionData.GameData.PlayerData, player.ToPlayerProtoMessage())
	}
	for _, entity := range entities {
		returnData.SessionData.GameData.EntityData = append(returnData.SessionData.GameData.EntityData, entity.ToEntityProtoMessage())
	}

	envelope := models.Envelope{
		Type:        models.MessageType_MESSAGE_TYPE_SESSION_DATA,
		MessageData: &returnData,
	}

	data, err := proto.Marshal(&envelope)
	if err != nil {
		s.logger.Printf("Error marshalling session data: %s", err.Error())
	}

	return data
}
