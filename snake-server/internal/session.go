package internal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const UpdateObjectsTime time.Duration = 1000 * time.Millisecond
const maxPlayers = 4

// Command types for the session actor
type addPlayerCmd struct {
	player   *Player
	resultCh chan error
}

type removePlayerCmd struct {
	playerID string
	resultCh chan error
}

type sendMessageCmd struct {
	playerID *string
	data     []byte
}

type getPlayersCmd struct {
	resultCh chan []*Player
}

type canJoinCmd struct {
	resultCh chan bool
}

type checkAllPlayersReadyCmd struct {
	resultCh chan bool
}

type updatePlayerDirectionCmd struct {
	playerID  string
	direction models.Direction
	resultCh  chan error
}

type updatePlayerStateCmd struct {
	playerID string
	state    models.PlayerState
	resultCh chan error
}

type getEntitiesCmd struct {
	resultCh chan []*Entity
}

type Session struct {
	SessionID string
	//Players              map[string]*Player
	PlayerSlots          PlayerSlots
	PlayerInputChannel   chan []byte
	Board                *Board
	UpdateEntitiesTicker *time.Ticker
	UpdateClientsTicker  *time.Ticker
	State                models.SessionState
	finishedChannel      chan string
	winningPlayerID      string

	// Actor pattern channels
	addPlayerCh       chan *addPlayerCmd
	removePlayerCh    chan *removePlayerCmd
	sendMessageCh     chan *sendMessageCmd
	getPlayersCh      chan *getPlayersCmd
	getEntitiesCh     chan *getEntitiesCmd
	canJoinCh         chan *canJoinCmd
	checkAllReadyCh   chan *checkAllPlayersReadyCmd
	updateDirectionCh chan *updatePlayerDirectionCmd
	updateStateCh     chan *updatePlayerStateCmd
	shutdownCh        chan struct{}

	// To handle shutting down the session
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	logger *log.Logger
}

func NewSession(finishedChannel chan string, logger *log.Logger) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	session := Session{
		SessionID: uuid.New().String(),
		//Players:              make(map[string]*Player, 0),
		Board:                &Board{},
		PlayerInputChannel:   make(chan []byte),
		UpdateEntitiesTicker: time.NewTicker(UpdateObjectsTime),
		State:                models.SessionState_SESSION_STATE_INITIALIZING,
		finishedChannel:      finishedChannel,
		PlayerSlots:          NewPlayerSlots(),

		// Initialize command channels
		addPlayerCh:       make(chan *addPlayerCmd),
		removePlayerCh:    make(chan *removePlayerCmd),
		sendMessageCh:     make(chan *sendMessageCmd, 100),
		getPlayersCh:      make(chan *getPlayersCmd),
		canJoinCh:         make(chan *canJoinCmd),
		checkAllReadyCh:   make(chan *checkAllPlayersReadyCmd),
		updateDirectionCh: make(chan *updatePlayerDirectionCmd),
		updateStateCh:     make(chan *updatePlayerStateCmd),
		shutdownCh:        make(chan struct{}, 1), // Buffered to allow non-blocking sends
		getEntitiesCh:     make(chan *getEntitiesCmd),

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
		wg:     sync.WaitGroup{},
	}
	logger.Printf("New session %s\n", session.SessionID)

	session.wg.Add(2)
	go func() {
		// The main actor goroutine
		defer session.wg.Done()
		session.run()
	}()
	go func() {
		// Reading messages players send
		defer session.wg.Done()
		session.ReadChannelMessages()
	}()

	return &session
}

// run is the central goroutine that owns the Players map
func (s *Session) run() {

	for {
		select {
		case cmd := <-s.addPlayerCh:
			cmd.resultCh <- s.addPlayerInternal(cmd.player)
			// Every time a player is added/deleted, update the other players
			players := s.PlayerSlots.GetPlayers()
			entities := s.Board.GetEntities()
			s.broadcastInternal(s.GetSessionData(players, entities))
		case cmd := <-s.removePlayerCh:
			cmd.resultCh <- s.removePlayerInternal(cmd.playerID)
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
		case cmd := <-s.sendMessageCh:
			s.sendMessageInternal(cmd.playerID, cmd.data)

		case cmd := <-s.getPlayersCh:
			players := s.PlayerSlots.GetPlayers()
			cmd.resultCh <- players

		case cmd := <-s.canJoinCh:
			cmd.resultCh <- (len(s.PlayerSlots.GetPlayers()) < maxPlayers && s.State == models.SessionState_SESSION_STATE_SETUP)

		case cmd := <-s.checkAllReadyCh:
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

			cmd.resultCh <- allReady

		case cmd := <-s.updateDirectionCh:
			player, ok := s.PlayerSlots.GetPlayer(cmd.playerID)
			if !ok {
				cmd.resultCh <- fmt.Errorf("Player ID %s doesn't exist in session", cmd.playerID)
			} else {
				player.UpdateDirection(cmd.direction)
				cmd.resultCh <- nil
			}

		case cmd := <-s.updateStateCh:
			player, ok := s.PlayerSlots.GetPlayer(cmd.playerID)
			if !ok {
				cmd.resultCh <- fmt.Errorf("Player ID %s doesn't exist in session", cmd.playerID)
			} else {
				player.UpdateState(cmd.state)
				cmd.resultCh <- nil
			}

		case cmd := <-s.getEntitiesCh:
			cmd.resultCh <- s.Board.GetEntities()

		case <-s.shutdownCh:
			// Whether the game was finished, or all players left, this session is over
			// Tell the session manager to delete this session
			// The actual session resources will cleanup after this goroutine is done
			s.initiateShutDown()
			s.finishedChannel <- s.SessionID
			return
		}
	}
}

// Internal methods - called only by run() goroutine

func (s *Session) addPlayerInternal(newPlayer *Player) error {

	if _, ok := s.PlayerSlots.GetPlayer(newPlayer.ID); ok {
		return fmt.Errorf("Player %s already exists in session", newPlayer.ID)
	}

	success := s.PlayerSlots.AddPlayer(newPlayer)
	if !success {
		return fmt.Errorf("could not add player %s to session %s", newPlayer.ID, s.SessionID)
	}
	s.logger.Printf("Player ID %s added to session: %s\n", newPlayer.ID, s.SessionID)

	go s.HandlePlayer(newPlayer)
	return nil
}

func (s *Session) removePlayerInternal(playerID string) error {
	player, ok := s.PlayerSlots.GetPlayer(playerID)
	if !ok {
		return fmt.Errorf("Player %s no longer exists in  when trying to remove", playerID)
	}

	err := s.PlayerSlots.RemovePlayer(playerID)
	if err != nil {
		fmt.Printf("error when attempting to shut down player %s: %v", playerID, err)
	}
	s.logger.Printf("Player ID %s deleted from session: %s\n", player.ID, s.SessionID)

	return nil
}

func (s *Session) broadcastInternal(data []byte) {
	// Create snapshot of players to avoid holding up the actor while doing I/O
	players := s.PlayerSlots.GetPlayers()

	// Do I/O outside of the actor select loop (in a goroutine for non-blocking)
	go func() {
		for _, player := range players {
			player.SendMessage(data)
		}
	}()
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

func (s *Session) AddPlayer(newPlayer *Player) error {
	// Use channel to add player to session
	resultCh := make(chan error)
	s.addPlayerCh <- &addPlayerCmd{
		player:   newPlayer,
		resultCh: resultCh,
	}
	return <-resultCh
}

func (s *Session) ReleasePlayer(playerID string) error {
	resultCh := make(chan error)
	s.removePlayerCh <- &removePlayerCmd{
		playerID: playerID,
		resultCh: resultCh,
	}
	return <-resultCh
}

func (s *Session) CanJoin() bool {
	// Before doing anything, check the context to see if it was cancelled.
	select {
	case <-s.ctx.Done():
		return false
	default:
	}

	resultCh := make(chan bool, 1)
	select {
	case s.canJoinCh <- &canJoinCmd{resultCh: resultCh}:
		select {
		case result := <-resultCh:
			return result
		case <-time.After(100 * time.Millisecond):
			return false // Timeout
		}
	case <-time.After(100 * time.Millisecond):
		return false // Can't send, session likely dead
	}
}

func (s *Session) StartSession() {
	playerCh := make(chan []*Player)
	s.getPlayersCh <- &getPlayersCmd{
		resultCh: playerCh,
	}
	players := <-playerCh

	if len(players) == 0 {
		return
	}

	s.Board.Initialize(s.PlayerSlots)
	s.State = models.SessionState_SESSION_STATE_RUNNING
	s.BroadcastMessage(s.GetSessionData(nil, nil))

	go s.UpdateState()
}

func (s *Session) HandlePlayer(player *Player) {
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
	// Now that we returned, was it a close error?
	if !websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
		// Log the error
		s.logger.Printf("Player ID %s closed unexpectadly\n", player.ID)
	} else {
		s.logger.Printf("Player ID %s closed connection\n", player.ID)
	}

	s.ReleasePlayer(player.ID)

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
	// for message := range s.PlayerInputChannel {
	// 	if err := s.HandleMessage(message); err != nil {
	// 		s.logger.Printf("Error reading message: %s : %s", string(message), err.Error())
	// 	}
	// }
}

func (s *Session) HandleMessage(message []byte) error {
	s.logger.Printf("Got message: current state is: %s\n", s.State)
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

	// Use channel to update player direction
	resultCh := make(chan error)
	s.updateDirectionCh <- &updatePlayerDirectionCmd{
		playerID:  input.PlayerID,
		direction: input.InputType,
		resultCh:  resultCh,
	}
	return <-resultCh
}

func (s *Session) HandlePlayerReadyUp(message []byte) error {
	// This should be 'ready' message
	input := &models.ReadyUp{}
	if err := proto.Unmarshal(message, input); err != nil {
		return err
	}

	// Use channel to update player state
	resultCh := make(chan error)
	s.updateStateCh <- &updatePlayerStateCmd{
		playerID: input.PlayerID,
		state:    models.PlayerState_STATE_READY,
		resultCh: resultCh,
	}
	if err := <-resultCh; err != nil {
		return err
	}

	sessionData := s.GetSessionData(nil, nil)
	s.BroadcastMessage(sessionData)

	if s.CheckAllPlayersReady() {
		s.StartSession()
	}

	return nil
}

func (s *Session) CheckAllPlayersReady() bool {
	resultCh := make(chan bool)
	s.checkAllReadyCh <- &checkAllPlayersReadyCmd{
		resultCh: resultCh,
	}
	return <-resultCh
}

func (s *Session) UpdateState() {
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
			s.BroadcastMessage(sessionData)
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
	s.sendMessageCh <- &sendMessageCmd{
		playerID: playerID,
		data:     data,
	}
}

func (s *Session) BroadcastMessage(data []byte) {
	// If there isn't a specific player ID in the SendMessageCmd, we will just broadcast to everyone
	s.sendMessageCh <- &sendMessageCmd{
		playerID: nil,
		data:     data,
	}
}

func (s *Session) GetPlayers() []*Player {
	// Get list of player IDs first via channel
	playersCh := make(chan []*Player)
	s.getPlayersCh <- &getPlayersCmd{
		resultCh: playersCh,
	}
	return <-playersCh
}

func (s *Session) GetEntities() []*Entity {
	entityCh := make(chan []*Entity)
	s.getEntitiesCh <- &getEntitiesCmd{
		resultCh: entityCh,
	}
	return <-entityCh
}

// Called from main actor goroutine: prepare the session for deletion from the sessionManager
func (s *Session) initiateShutDown() {
	s.cancel() // SessionManager can read the context to know this session is cancelled (done)
	s.UpdateEntitiesTicker.Stop()
}

// Called by the session manager, actually clean up the session
func (s *Session) DeleteSession() {
	s.logger.Printf("Shutting down session: %s", s.SessionID)
	// Wait for run and input goroutines to finish
	s.wg.Wait()

	for _, player := range s.PlayerSlots.GetPlayers() {
		// We are still in the main goroutine, release the players
		if err := s.removePlayerInternal(player.ID); err != nil {
			// even if it errors, need to continue
			s.logger.Printf("error removing player %s:%v", player.ID, err)
		}
	}

	close(s.PlayerInputChannel)
}

// func (s *Session) DeleteSessionResources() {
// 	// Don't use GetPlayers() since the actor goroutine may have already shut down
// 	// Access players map directly
// 	s.logger.Printf("Deleting session resources for session %s", s.SessionID)

// 	// Shut down all player connections directly (not through channels)
// 	for _, player := range s.Players {
// 		s.logger.Printf("Shutting down player %s", player.ID)
// 		err := player.ShutDownPlayer()
// 		if err != nil {
// 			s.logger.Printf("Error shutting down player ID %s: %s", player.ID, err.Error())
// 		}
// 	}

// 	// Clear the players map
// 	s.Players = make(map[string]*Player)

// 	// Close channels if not already closed
// 	// Note: PlayerInputChannel and shutdownCh might already be closed
// 	// We need to be careful about closing them again
// 	defer func() {
// 		if r := recover(); r != nil {
// 			s.logger.Printf("Recovered from panic while closing channels: %v", r)
// 		}
// 	}()

// 	// Try to close PlayerInputChannel
// 	select {
// 	case <-s.PlayerInputChannel:
// 		// Channel already closed, do nothing
// 	default:
// 		close(s.PlayerInputChannel)
// 	}
// }

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

	s.logger.Printf("Sending %d entities", len(entities))
	return data
}
