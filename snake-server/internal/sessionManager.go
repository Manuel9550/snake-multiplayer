package internal

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Manuel9550/snake-multiplayer/snake-server/gen/models"
	"github.com/gorilla/websocket"
)

type SessionManager struct {
	sessions    map[string]*Session
	sessionsMu  sync.Mutex
	doneChannel chan string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*Session),
		sessionsMu:  sync.Mutex{},
		doneChannel: make(chan string),
	}
}

func (s *SessionManager) HandleConnection(upgrader *websocket.Upgrader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Error upgrading:", err)
			return
		}

		newPlayer := NewPlayer(conn)

		s.sessionsMu.Lock()
		defer s.sessionsMu.Unlock()

		sessionToAddPlayerTo := s.OpenSession()
		if sessionToAddPlayerTo != nil {
			err := sessionToAddPlayerTo.AddPlayer(newPlayer)
			if err != nil {
				fmt.Println("Error upgrading:", err)
			}
			return
		}

		newSession := NewSession(s.doneChannel)
		newSession.State = models.SessionState_SESSION_STATE_SETUP
		err = newSession.AddPlayer(newPlayer)
		if err != nil {
			fmt.Println("Error adding player to session:", err)
			return
		}

		err = s.AddSession(newSession)
		if err != nil {
			fmt.Println("Error adding session to session manager:", err)
			return
		}
	}
}

func (s *SessionManager) AddSession(session *Session) error {
	_, ok := s.sessions[session.SessionID]
	if ok {
		return fmt.Errorf("session %s already exists", session.SessionID)
	}

	s.sessions[session.SessionID] = session

	return nil
}

func (s *SessionManager) DeleteSession(sessionID string) error {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	foundSession, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %s does not exists", sessionID)
	}

	foundSession.DeleteSessionResources()
	copiedMap := make(map[string]*Session, len(s.sessions)-1)
	for key, value := range s.sessions {
		if key != sessionID {
			copiedMap[key] = value
		}
	}
	s.sessions = copiedMap
	fmt.Printf("Deleted session %s\n", sessionID)

	return nil
}

func (s *SessionManager) MonitorSessions() {
	for currentMessage := range s.doneChannel {
		s.DeleteSession(currentMessage)
	}
}

func (s *SessionManager) OpenSession() *Session {
	for _, session := range s.sessions {
		if session.CanJoin() {
			return session
		}
	}

	return nil
}
