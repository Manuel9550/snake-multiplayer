package main

import (
	"fmt"
	"net/http"

	"github.com/Manuel9550/snake-multiplayer/snake-server/internal"
	"github.com/gorilla/websocket"
)

// Upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: validate the origin in production
		return true
	},
}

func main() {
	sessionManager := internal.NewSessionManager()

	go sessionManager.MonitorSessions()
	http.HandleFunc("/ws", sessionManager.HandleConnection(&upgrader))
	fmt.Println("WebSocket server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}

}
