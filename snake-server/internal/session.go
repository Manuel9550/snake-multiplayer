package internal

import (
	"fmt"
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
	state    models.State
	resultCh chan error
}

type Session struct {
	SessionID            string
	Entities             map[string]*Entity
	Players              map[string]*Player
	PlayerInputChannel   chan []byte
	Board                *Board
	UpdateEntitiesTicker *time.Ticker
	UpdateClientsTicker  *time.Ticker
	State                models.SessionState
	finishedChannel      chan string

	// Actor pattern channels
	addPlayerCh       chan *addPlayerCmd
	removePlayerCh    chan *removePlayerCmd
	sendMessageCh     chan *sendMessageCmd
	getPlayersCh      chan *getPlayersCmd
	canJoinCh         chan *canJoinCmd
	checkAllReadyCh   chan *checkAllPlayersReadyCmd
	updateDirectionCh chan *updatePlayerDirectionCmd
	updateStateCh     chan *updatePlayerStateCmd
	shutdownCh        chan struct{}
}

func NewSession(finishedChannel chan string) *Session {
	session := Session{
		SessionID:            uuid.New().String(),
		Entities:             make(map[string]*Entity, 0),
		Players:              make(map[string]*Player, 0),
		Board:                &Board{},
		PlayerInputChannel:   make(chan []byte),
		UpdateEntitiesTicker: time.NewTicker(UpdateObjectsTime),
		State:                models.SessionState_SESSION_STATE_INITIALIZING,
		finishedChannel:      finishedChannel,

		// Initialize command channels
		addPlayerCh:       make(chan *addPlayerCmd),
		removePlayerCh:    make(chan *removePlayerCmd),
		sendMessageCh:     make(chan *sendMessageCmd, 100),
		getPlayersCh:      make(chan *getPlayersCmd),
		canJoinCh:         make(chan *canJoinCmd),
		checkAllReadyCh:   make(chan *checkAllPlayersReadyCmd),
		updateDirectionCh: make(chan *updatePlayerDirectionCmd),
		updateStateCh:     make(chan *updatePlayerStateCmd),
		shutdownCh:        make(chan struct{}),
	}
	fmt.Printf("New session %s\n", session.SessionID)

	go session.run() // Start the actor goroutine
	go session.ReadChannelMessages()
	return &session
}

// run is the central goroutine that owns the Players map
func (s *Session) run() {
	for {
		select {
		case cmd := <-s.addPlayerCh:
			cmd.resultCh <- s.addPlayerInternal(cmd.player)

		case cmd := <-s.removePlayerCh:
			cmd.resultCh <- s.removePlayerInternal(cmd.playerID)

		case cmd := <-s.sendMessageCh:
			s.sendMessageInternal(cmd.playerID, cmd.data)

		case cmd := <-s.getPlayersCh:
			players := make([]*Player, 0, len(s.Players))
			for _, p := range s.Players {
				players = append(players, p)
			}
			cmd.resultCh <- players

		case cmd := <-s.canJoinCh:
			fmt.Printf("Len players: %d, current state: %v", len(s.Players), s.State)
			cmd.resultCh <- (len(s.Players) < maxPlayers && s.State == models.SessionState_SESSION_STATE_SETUP)

		case cmd := <-s.checkAllReadyCh:
			allReady := true
			playerCount := 0
			for _, player := range s.Players {
				playerCount += 1
				if player.State != models.State_STATE_READY {
					allReady = false
					break
				}
			}
			if playerCount < 2 {
				allReady = false
			}

			cmd.resultCh <- allReady

		case cmd := <-s.updateDirectionCh:
			player, ok := s.Players[cmd.playerID]
			if !ok {
				cmd.resultCh <- fmt.Errorf("Player ID %s doesn't exist in session", cmd.playerID)
			} else {
				player.UpdateDirection(cmd.direction)
				cmd.resultCh <- nil
			}

		case cmd := <-s.updateStateCh:
			player, ok := s.Players[cmd.playerID]
			if !ok {
				cmd.resultCh <- fmt.Errorf("Player ID %s doesn't exist in session", cmd.playerID)
			} else {
				player.UpdateState(cmd.state)
				cmd.resultCh <- nil
			}

		case <-s.shutdownCh:
			return
		}
	}
}

// Internal methods - called only by run() goroutine, no locking needed

func (s *Session) addPlayerInternal(newPlayer *Player) error {
	if _, ok := s.Players[newPlayer.ID]; ok {
		return fmt.Errorf("Player %s already exists in session", newPlayer.ID)
	}

	// Assign color based on player count
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

	newPlayer.State = models.State_STATE_CONNECTED
	s.Players[newPlayer.ID] = newPlayer
	fmt.Printf("Player ID %s added to session: %s\n", newPlayer.ID, s.SessionID)

	go s.HandlePlayer(newPlayer)
	return nil
}

func (s *Session) removePlayerInternal(playerID string) error {
	player, ok := s.Players[playerID]
	if !ok {
		return fmt.Errorf("Player %s no longer exists in session", playerID)
	}

	player.ShutDownPlayer()
	delete(s.Players, playerID)
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

func (s *Session) broadcastInternal(data []byte) {
	// Create snapshot of players to avoid holding up the actor while doing I/O
	players := make([]*Player, 0, len(s.Players))
	for _, player := range s.Players {
		players = append(players, player)
	}

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

	player, ok := s.Players[*playerID]
	if !ok || player == nil {
		fmt.Printf("Error finding player with ID %s\n", *playerID)
		return
	}

	// Do I/O in goroutine to avoid blocking the actor
	go func() {
		player.SendMessage(data)
	}()
}

// Public API methods - send commands via channels

func (s *Session) AddPlayer(newPlayer *Player) error {
	// Send player data message before adding to session
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

	newPlayer.SendMessage(data)

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
	resultCh := make(chan bool)
	s.canJoinCh <- &canJoinCmd{
		resultCh: resultCh,
	}
	return <-resultCh
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

	s.Board.Initialize(players)

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
		return
	}

	s.BroadcastMessage(data)

	go s.UpdateState()
}

func (s *Session) HandlePlayer(player *Player) {
	// If conn is nil, this is a test scenario - don't auto-remove the player
	if player.conn == nil {
		// Still broadcast player states for testing purposes
		s.BroadcastPlayerStates()
		return
	}

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
	s.SendMessage(&player.ID, data)

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
	// No need for mutex - ReleasePlayer handles its own channel-based synchronization
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
	fmt.Printf("In HandlePlayerReadyUp\n")
	input := &models.PlayerState{}
	if err := proto.Unmarshal(message, input); err != nil {
		return err
	}

	if input.State != models.State_STATE_READY {
		// Intentionally ignore messages that aren't readying up
		fmt.Printf("Ignoring non ready up message, got: %v", input.State)
		return nil
	}

	// Use channel to update player state
	resultCh := make(chan error)
	s.updateStateCh <- &updatePlayerStateCmd{
		playerID: input.PlayerID,
		state:    input.State,
		resultCh: resultCh,
	}
	if err := <-resultCh; err != nil {
		return err
	}

	s.BroadcastPlayerStates()

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
	for range s.UpdateEntitiesTicker.C {
		players := s.GetPlayers()
		s.UpdateBoard(players)
		gameDone, winningPlayer := s.CheckGameOver(players)
		if gameDone {
			s.EndSession(winningPlayer)
			return
		}
		s.UpdateClients(players)
	}
}

func (s *Session) EndSession(winningPlayer *Player) {
	s.State = models.SessionState_SESSION_STATE_DONE
	playerID := ""
	if winningPlayer != nil {
		playerID = winningPlayer.ID
	}
	envelope := models.Envelope{
		Type: models.MessageType_MESSAGE_TYPE_GAME_OVER,
		MessageData: &models.Envelope_GameOver{
			GameOver: &models.GameOver{
				SessionID:       s.SessionID,
				WinningPlayerID: playerID,
			},
		},
	}
	data, err := proto.Marshal(&envelope)
	if err != nil {
		fmt.Println("Error marshalling message:", err)
	}

	s.BroadcastMessage(data)

}

func (s *Session) UpdateBoard(players []*Player) {
	for _, player := range players {
		if player.State != models.State_STATE_DONE {
			s.Board.Update(player)
		}
	}
}

func (s *Session) CheckGameOver(players []*Player) (bool, *Player) {
	activePlayers := 0
	var activePlayer *Player
	for _, player := range players {
		if player.State != models.State_STATE_PLAYING {
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

func (s *Session) UpdateClients(players []*Player) {
	gameData := models.GameData{
		PlayerData: []*models.Player{},
		EntityData: []*models.Entity{},
	}
	for _, player := range players {
		gameData.PlayerData = append(gameData.PlayerData, player.ToPlayerProtoMessage())
	}
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

func (s *Session) DeleteSessionResources() {
	players := s.GetPlayers()

	// Release all players
	for _, player := range players {
		err := s.ReleasePlayer(player.ID)
		if err != nil {
			fmt.Printf("Error releasing player ID %s:%s", player.ID, err.Error())
		}
	}

	close(s.PlayerInputChannel)
	close(s.shutdownCh) // Shut down the actor goroutine
}

func (s *Session) PlayerStates() models.PlayerStates {
	// Get the players, before getting their state
	playersCh := make(chan []*Player)
	s.getPlayersCh <- &getPlayersCmd{
		resultCh: playersCh,
	}
	players := <-playersCh

	returnValues := models.PlayerStates{
		PlayerStates: make([]*models.PlayerState, 0, len(players)),
	}
	for _, player := range players {
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
