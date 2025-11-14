package main

import (
	"io"
	"log"
	"net/http"
	"os"

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
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close() // Ensure the log file is closed when the program exits

	// Create a MultiWriter that writes to both os.Stdout and the logFile
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create a new logger that uses the MultiWriter
	// log.LstdFlags adds standard flags like date and time to log entries
	logger := log.New(multiWriter, "APP: ", log.LstdFlags)

	// Try to load the pre-made layouts
	err = internal.InitializePremadelayouts("./../images")
	if err != nil {
		logger.Printf("Failed to load Premade Layouts: %v", err)
	}

	sessionManager := internal.NewSessionManager(logger)
	go sessionManager.MonitorSessions()
	http.HandleFunc("/ws", sessionManager.HandleConnection(&upgrader))
	logger.Printf("WebSocket server started on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Printf("Error starting server: %s", err.Error())
	}

}
