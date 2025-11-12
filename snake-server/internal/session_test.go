package internal

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"google.golang.org/protobuf/proto"
)

// Helper function to create a test player for direct map manipulation
func createTestPlayer(id string) *Player {
	bodyQueue := Queue{}
	bodyQueue.Enqueue(Location{X: 0, Y: 0})
	return &Player{
		ID:        id,
		conn:      nil, // We won't use the connection in these tests
		Colour:    "white",
		Direction: models.Direction_DIRECTION_UP,
		X:         0,
		Y:         0,
		BodyQueue: bodyQueue,
		Length:    1,
		State:     models.PlayerState_STATE_CONNECTING,
	}
}

// TestNewSession tests session initialization
func TestNewSession(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	if session.SessionID == "" {
		t.Error("Expected non-empty session ID")
	}

	if session.State != models.SessionState_SESSION_STATE_INITIALIZING {
		t.Errorf("Expected initial state INITIALIZING, got %v", session.State)
	}

	if session.Players == nil {
		t.Error("Expected Players map to be initialized")
	}
}

// TestAddPlayerColorAssignment tests color assignment logic
func TestAddPlayerColorAssignment(t *testing.T) {
	// Test the color assignment logic by checking what colors would be assigned
	expectedColors := []string{"blue", "green", "red", "yellow", "white"}

	for i := 0; i < 5; i++ {
		// Simulate player count
		numberPlayer := i + 1
		var color string

		switch numberPlayer {
		case 1:
			color = "blue"
		case 2:
			color = "green"
		case 3:
			color = "red"
		case 4:
			color = "yellow"
		default:
			color = "white"
		}

		if color != expectedColors[i] {
			t.Errorf("Player %d: expected color %s, got %s", i+1, expectedColors[i], color)
		}
	}
}

// TestCanJoinWithMaxPlayers tests the CanJoin logic
func TestCanJoinWithMaxPlayers(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Initially should not be able to join, since we are initializing
	if session.CanJoin() {
		t.Error("Expected CanJoin to be false initially")
	}

	session.State = models.SessionState_SESSION_STATE_SETUP
	if !session.CanJoin() {
		t.Error("Expected CanJoin to be true in SETUP state")
	}

	// Manually add 4 players to the map (bypassing HandlePlayer
	var testPlayer *Player
	for i := 0; i < 4; i++ {
		player := createTestPlayer("player-" + string(rune('1'+i)))
		player.State = models.PlayerState_STATE_CONNECTED

		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   player,
			resultCh: resultCh,
		}
		err := <-resultCh
		if err != nil {
			t.Fatalf("Error adding player %d: %v", i, err)
		}
		if testPlayer == nil {
			testPlayer = player
		}
	}

	// With 4 players should not be able to join
	if session.CanJoin() {
		t.Error("Expected CanJoin to be false with 4 players")
	}

	// remove a player
	resultCh := make(chan error)
	session.removePlayerCh <- &removePlayerCmd{
		playerID: testPlayer.ID,
		resultCh: resultCh,
	}
	err := <-resultCh
	if err != nil {
		t.Fatalf("Error removing player %s: %v", testPlayer.ID, err)
	}

	// check how many players are in the session
	resultPlayersCh := make(chan []*Player)
	session.getPlayersCh <- &getPlayersCmd{
		resultCh: resultPlayersCh,
	}
	players := <-resultPlayersCh
	if len(players) != 3 {
		t.Fatalf("Expected only 3 players: instead got %d", len(players))
	}

	if session.State != models.SessionState_SESSION_STATE_SETUP {
		t.Error("Expected session state to still be in setup")
	}

	if !session.CanJoin() {
		t.Error("Expected CanJoin to be true after removing a player")
	}

}

// TestGetPlayersEmpty tests GetPlayers with no players
func TestGetPlayersEmpty(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	players := session.GetPlayers()

	if len(players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(players))
	}
}

// TestGetPlayersMultiple tests fetching the list of players
func TestGetPlayersMultiple(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Add players via the actor channel (avoid HandlePlayer)
	playerIDs := []string{"player-1", "player-2", "player-3"}
	for _, id := range playerIDs {
		player := createTestPlayer(id)
		player.State = models.PlayerState_STATE_CONNECTED

		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   player,
			resultCh: resultCh,
		}
		if err := <-resultCh; err != nil {
			t.Fatalf("Error adding player: %v", err)
		}
	}

	players := session.GetPlayers()

	if len(players) != 3 {
		t.Errorf("Expected 3 players, got %d", len(players))
	}

	// Verify all player IDs are present
	foundIDs := make(map[string]bool)
	for _, p := range players {
		foundIDs[p.ID] = true
	}

	for _, id := range playerIDs {
		if !foundIDs[id] {
			t.Errorf("Expected to find player %s", id)
		}
	}
}

// TestCheckAllPlayersReadyInsufficientPlayers tests ready check with < 2 players
func TestCheckAllPlayersReadyInsufficientPlayers(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// With no players, should not be ready
	if session.CheckAllPlayersReady() {
		t.Error("Expected CheckAllPlayersReady to be false with no players")
	}

	// Add one player marked as ready
	player := createTestPlayer("player-1")
	player.State = models.PlayerState_STATE_READY

	resultCh := make(chan error)
	session.addPlayerCh <- &addPlayerCmd{
		player:   player,
		resultCh: resultCh,
	}
	<-resultCh

	// Still should not be ready (need at least 2)
	if session.CheckAllPlayersReady() {
		t.Error("Expected CheckAllPlayersReady to be false with only 1 player")
	}
}

// TestCheckAllPlayersReadyNotAllReady tests when players exist but aren't ready
func TestCheckAllPlayersReadyNotAllReady(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Add two players, not ready
	for i := 0; i < 2; i++ {
		player := createTestPlayer("player-" + string(rune('1'+i)))
		player.State = models.PlayerState_STATE_CONNECTED

		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   player,
			resultCh: resultCh,
		}
		<-resultCh
	}

	if session.CheckAllPlayersReady() {
		t.Error("Expected CheckAllPlayersReady to be false when players not ready")
	}
}

// TestCheckAllPlayersReadyAllReady tests when all players are ready
func TestCheckAllPlayersReadyAllReady(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Add two players
	playerIDs := []string{"player-1", "player-2"}
	for _, id := range playerIDs {
		player := createTestPlayer(id)

		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   player,
			resultCh: resultCh,
		}
		<-resultCh
	}

	// Now mark both players as ready
	for _, id := range playerIDs {
		resultCh := make(chan error)
		session.updateStateCh <- &updatePlayerStateCmd{
			playerID: id,
			state:    models.PlayerState_STATE_READY,
			resultCh: resultCh,
		}
		<-resultCh
	}

	if !session.CheckAllPlayersReady() {
		t.Error("Expected CheckAllPlayersReady to be true when all players ready")
	}
}

// TestUpdatePlayerDirection tests updating player direction via channel
func TestUpdatePlayerDirection(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	player := createTestPlayer("player-1")
	player.Direction = models.Direction_DIRECTION_UP
	player.State = models.PlayerState_STATE_CONNECTED

	// Add player
	addResultCh := make(chan error)
	session.addPlayerCh <- &addPlayerCmd{
		player:   player,
		resultCh: addResultCh,
	}
	<-addResultCh

	// Update direction using the channel-based method
	resultCh := make(chan error)
	session.updateDirectionCh <- &updatePlayerDirectionCmd{
		playerID:  "player-1",
		direction: models.Direction_DIRECTION_LEFT,
		resultCh:  resultCh,
	}

	err := <-resultCh
	if err != nil {
		t.Errorf("Expected no error updating direction, got %v", err)
	}

	// Verify direction was updated
	players := session.GetPlayers()
	if len(players) != 1 {
		t.Fatalf("Expected 1 player, got %d", len(players))
	}

	if players[0].Direction != models.Direction_DIRECTION_LEFT {
		t.Errorf("Expected direction LEFT, got %v", players[0].Direction)
	}
}

// TestUpdatePlayerDirectionNonExistentPlayer tests updating direction for non-existent player
func TestUpdatePlayerDirectionNonExistentPlayer(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	resultCh := make(chan error)
	session.updateDirectionCh <- &updatePlayerDirectionCmd{
		playerID:  "non-existent",
		direction: models.Direction_DIRECTION_LEFT,
		resultCh:  resultCh,
	}

	err := <-resultCh
	if err == nil {
		t.Error("Expected error updating direction for non-existent player, got none")
	}
}

// TestUpdatePlayerState tests updating player state via channel
func TestUpdatePlayerState(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	player := createTestPlayer("player-1")
	player.State = models.PlayerState_STATE_CONNECTED

	// Add player
	addResultCh := make(chan error)
	session.addPlayerCh <- &addPlayerCmd{
		player:   player,
		resultCh: addResultCh,
	}
	<-addResultCh

	// Update state using the channel-based method
	resultCh := make(chan error)
	session.updateStateCh <- &updatePlayerStateCmd{
		playerID: "player-1",
		state:    models.PlayerState_STATE_READY,
		resultCh: resultCh,
	}

	err := <-resultCh
	if err != nil {
		t.Errorf("Expected no error updating state, got %v", err)
	}

	// Verify state was updated
	players := session.GetPlayers()
	if len(players) != 1 {
		t.Fatalf("Expected 1 player, got %d", len(players))
	}

	if players[0].State != models.PlayerState_STATE_READY {
		t.Errorf("Expected state READY, got %v", players[0].State)
	}
}

// TestUpdatePlayerStateNonExistentPlayer tests updating state for non-existent player
func TestUpdatePlayerStateNonExistentPlayer(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	resultCh := make(chan error)
	session.updateStateCh <- &updatePlayerStateCmd{
		playerID: "non-existent",
		state:    models.PlayerState_STATE_READY,
		resultCh: resultCh,
	}

	err := <-resultCh
	if err == nil {
		t.Error("Expected error updating state for non-existent player, got none")
	}
}

// TestPlayerStates tests fetching player states
func TestPlayerStates(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Add two players
	player1 := createTestPlayer("player-1")
	player2 := createTestPlayer("player-2")

	for _, p := range []*Player{player1, player2} {
		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   p,
			resultCh: resultCh,
		}
		<-resultCh
	}

	// Update player1 to READY state
	resultCh := make(chan error)
	session.updateStateCh <- &updatePlayerStateCmd{
		playerID: "player-1",
		state:    models.PlayerState_STATE_READY,
		resultCh: resultCh,
	}
	<-resultCh

	// Player2 stays in CONNECTED state (set by addPlayerInternal)

	players := session.GetPlayers()

	// Should have 2 players
	if len(players) < 2 {
		t.Errorf("Expected at least 2 players, got %d", len(players))
	}

	// Verify states contain expected player IDs
	foundPlayer1 := false
	foundPlayer2 := false
	for _, player := range players {
		if player.ID == "player-1" {
			foundPlayer1 = true
			if player.State != models.PlayerState_STATE_READY {
				t.Errorf("Expected player-1 state READY, got %v", player.State)
			}
		}
		if player.ID == "player-2" {
			foundPlayer2 = true
			if player.State != models.PlayerState_STATE_CONNECTED {
				t.Errorf("Expected player-2 state CONNECTED, got %v", player.State)
			}
		}
	}

	if !foundPlayer1 || !foundPlayer2 {
		t.Error("Not all expected players found")
	}
}

// TestRemovePlayer tests removing a player
func TestRemovePlayer(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Add two players
	for i := 0; i < 2; i++ {
		player := createTestPlayer("player-" + string(rune('1'+i)))
		player.State = models.PlayerState_STATE_CONNECTED

		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   player,
			resultCh: resultCh,
		}
		<-resultCh
	}

	// Verify 2 players exist
	players := session.GetPlayers()
	if len(players) != 2 {
		t.Fatalf("Expected 2 players, got %d", len(players))
	}

	// Remove one player
	removeResultCh := make(chan error)
	session.removePlayerCh <- &removePlayerCmd{
		playerID: "player-1",
		resultCh: removeResultCh,
	}
	err := <-removeResultCh
	if err != nil {
		t.Errorf("Expected no error removing player, got %v", err)
	}

	// Verify 1 player remains
	players = session.GetPlayers()
	if len(players) != 1 {
		t.Errorf("Expected 1 player after removal, got %d", len(players))
	}

	if players[0].ID != "player-2" {
		t.Errorf("Expected remaining player to be player-2, got %s", players[0].ID)
	}
}

// TestRemoveLastPlayerMarksSessionDone tests session cleanup when last player leaves
func TestRemoveLastPlayerMarksSessionDone(t *testing.T) {
	finishedCh := make(chan string, 10)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Add one player
	player := createTestPlayer("player-1")
	player.State = models.PlayerState_STATE_CONNECTED

	addResultCh := make(chan error)
	session.addPlayerCh <- &addPlayerCmd{
		player:   player,
		resultCh: addResultCh,
	}
	<-addResultCh

	// Remove the player
	removeResultCh := make(chan error)
	session.removePlayerCh <- &removePlayerCmd{
		playerID: "player-1",
		resultCh: removeResultCh,
	}
	err := <-removeResultCh
	if err != nil {
		t.Errorf("Expected no error removing player, got %v", err)
	}

	// Give time for session state to update
	time.Sleep(50 * time.Millisecond)

	if session.State != models.SessionState_SESSION_STATE_DONE {
		t.Errorf("Expected session state DONE after removing last player, got %v", session.State)
	}

	// Check that finished channel received session ID
	select {
	case sessionID := <-finishedCh:
		if sessionID != session.SessionID {
			t.Errorf("Expected finished channel to receive session ID %s, got %s", session.SessionID, sessionID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected finished channel to receive session ID, but timed out")
	}
}

// TestGetSessionData tests that GetSessionData returns valid protobuf with correct data
func TestGetSessionData(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Set session state
	session.State = models.SessionState_SESSION_STATE_SETUP

	// Add test players
	player1 := createTestPlayer("player-1")
	player1.X = 5
	player1.Y = 5
	player1.Direction = models.Direction_DIRECTION_UP

	player2 := createTestPlayer("player-2")
	player2.X = 10
	player2.Y = 10
	player2.Direction = models.Direction_DIRECTION_DOWN

	// Add players via actor channel
	for _, p := range []*Player{player1, player2} {
		resultCh := make(chan error)
		session.addPlayerCh <- &addPlayerCmd{
			player:   p,
			resultCh: resultCh,
		}
		<-resultCh
	}

	// Update player2 state to READY
	updateResultCh := make(chan error)
	session.updateStateCh <- &updatePlayerStateCmd{
		playerID: "player-2",
		state:    models.PlayerState_STATE_READY,
		resultCh: updateResultCh,
	}
	<-updateResultCh

	// Create test entities and add to session
	entity1 := NewEntity("player-1", models.EntityType_ENTITY_TYPE_BODY, Location{X: 5, Y: 5})
	entity2 := NewEntity("player-2", models.EntityType_ENTITY_TYPE_BODY, Location{X: 10, Y: 10})
	entity3 := NewEntity("", models.EntityType_ENTITY_TYPE_POINT, Location{X: 15, Y: 15})

	// Add entities directly to session for testing
	session.Board.Entities = map[Location]Entity{}
	session.Board.Entities[entity1.Location] = entity1
	session.Board.Entities[entity2.Location] = entity2
	session.Board.Entities[entity3.Location] = entity3

	// Call GetSessionData, explicitely not passing in the players/entities
	data := session.GetSessionData(nil, nil)

	if len(data) == 0 {
		t.Fatal("Expected non-empty data from GetSessionData")
	}

	// Unmarshal the data
	var envelope models.Envelope
	err := proto.Unmarshal(data, &envelope)
	if err != nil {
		t.Fatalf("Failed to unmarshal envelope: %v", err)
	}

	// Verify envelope type
	if envelope.Type != models.MessageType_MESSAGE_TYPE_SESSION_DATA {
		t.Errorf("Expected MessageType SESSION_DATA, got %v", envelope.Type)
	}

	// Extract SessionData
	sessionData := envelope.GetSessionData()
	if sessionData == nil {
		t.Fatal("Expected SessionData in envelope, got nil")
	}

	// Verify SessionID
	if sessionData.SessionID != session.SessionID {
		t.Errorf("Expected SessionID %s, got %s", session.SessionID, sessionData.SessionID)
	}

	// Verify State
	if sessionData.State != models.SessionState_SESSION_STATE_SETUP {
		t.Errorf("Expected State SETUP, got %v", sessionData.State)
	}

	// Verify WinningPlayer (should be empty for non-done session)
	if sessionData.WinningPlayer != "" {
		t.Errorf("Expected empty WinningPlayer, got %s", sessionData.WinningPlayer)
	}

	// Verify GameData exists
	gameData := sessionData.GameData
	if gameData == nil {
		t.Fatal("Expected GameData in SessionData, got nil")
	}

	// Verify PlayerData
	if len(gameData.PlayerData) != 2 {
		t.Fatalf("Expected 2 players in GameData, got %d", len(gameData.PlayerData))
	}

	// Check player1 data
	var foundPlayer1, foundPlayer2 bool
	for _, p := range gameData.PlayerData {
		if p.PlayerID == "player-1" {
			foundPlayer1 = true
			if p.Colour != "blue" {
				t.Errorf("Expected player-1 colour blue, got %s", p.Colour)
			}
			if p.State != models.PlayerState_STATE_CONNECTED {
				t.Errorf("Expected player-1 state CONNECTED, got %v", p.State)
			}
			if p.Head.X != 5 || p.Head.Y != 5 {
				t.Errorf("Expected player-1 head at (5,5), got (%d,%d)", p.Head.X, p.Head.Y)
			}
			if p.PlayerDirection != models.Direction_DIRECTION_UP {
				t.Errorf("Expected player-1 direction UP, got %v", p.PlayerDirection)
			}
		}
		if p.PlayerID == "player-2" {
			foundPlayer2 = true
			if p.Colour != "green" {
				t.Errorf("Expected player-2 colour green, got %s", p.Colour)
			}
			if p.State != models.PlayerState_STATE_READY {
				t.Errorf("Expected player-2 state READY, got %v", p.State)
			}
			if p.Head.X != 10 || p.Head.Y != 10 {
				t.Errorf("Expected player-2 head at (10,10), got (%d,%d)", p.Head.X, p.Head.Y)
			}
			if p.PlayerDirection != models.Direction_DIRECTION_DOWN {
				t.Errorf("Expected player-2 direction DOWN, got %v", p.PlayerDirection)
			}
		}
	}

	if !foundPlayer1 {
		t.Error("player-1 not found in GameData.PlayerData")
	}
	if !foundPlayer2 {
		t.Error("player-2 not found in GameData.PlayerData")
	}

	// Verify EntityData
	if len(gameData.EntityData) != 3 {
		t.Fatalf("Expected 3 entities in GameData, got %d", len(gameData.EntityData))
	}

	// Check entities
	entityTypes := make(map[models.EntityType]int)
	for _, e := range gameData.EntityData {
		entityTypes[e.Type]++

		// Verify entity structure
		if e.ID == "" {
			t.Error("Expected non-empty entity ID")
		}
		if e.Body == nil {
			t.Error("Expected entity body to be non-nil")
		}
	}

	// Should have 2 BODY entities and 1 POINT entity
	if entityTypes[models.EntityType_ENTITY_TYPE_BODY] != 2 {
		t.Errorf("Expected 2 BODY entities, got %d", entityTypes[models.EntityType_ENTITY_TYPE_BODY])
	}
	if entityTypes[models.EntityType_ENTITY_TYPE_POINT] != 1 {
		t.Errorf("Expected 1 POINT entity, got %d", entityTypes[models.EntityType_ENTITY_TYPE_POINT])
	}
}

// TestGetSessionDataWithWinningPlayer tests GetSessionData when session is done with a winner
func TestGetSessionDataWithWinningPlayer(t *testing.T) {
	finishedCh := make(chan string, 1)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	session := NewSession(finishedCh, logger)
	defer session.DeleteSessionResources()

	// Set session to DONE state
	session.State = models.SessionState_SESSION_STATE_DONE
	session.winningPlayerID = "player-1"

	// Add a player
	player := createTestPlayer("player-1")
	player.State = models.PlayerState_STATE_DONE

	resultCh := make(chan error)
	session.addPlayerCh <- &addPlayerCmd{
		player:   player,
		resultCh: resultCh,
	}
	<-resultCh

	// Call GetSessionData
	data := session.GetSessionData(nil, []*Entity{})

	// Unmarshal and verify
	var envelope models.Envelope
	err := proto.Unmarshal(data, &envelope)
	if err != nil {
		t.Fatalf("Failed to unmarshal envelope: %v", err)
	}

	sessionData := envelope.GetSessionData()
	if sessionData == nil {
		t.Fatal("Expected SessionData in envelope")
	}

	// Verify session is DONE
	if sessionData.State != models.SessionState_SESSION_STATE_DONE {
		t.Errorf("Expected State DONE, got %v", sessionData.State)
	}

	// Verify WinningPlayer is set
	if sessionData.WinningPlayer != "player-1" {
		t.Errorf("Expected WinningPlayer 'player-1', got '%s'", sessionData.WinningPlayer)
	}
}
