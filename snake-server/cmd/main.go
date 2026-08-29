package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Manuel9550/snake-multiplayer/snake-server/internal"
	"github.com/gorilla/websocket"
)

const logDirectory = "./logs"
const frontEndDirectory = "./dist"
const testEnv = "TEST_ENV"

// Upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "https://snake-online.mannymakes.ca"
	},
}

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	err := os.MkdirAll(logDirectory, 0755)
	if err != nil {
		log.Fatalf("Error creating directory: %v", err)
	}

	fileName := fmt.Sprintf("%s.log", time.Now().Format("2006-01-02 15-04-05"))

	logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create a MultiWriter that writes to both os.Stdout and the logFile
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create a new logger that uses the MultiWriter
	// log.LstdFlags adds standard flags like date and time to log entries
	logger := log.New(multiWriter, "APP: ", log.LstdFlags)

	// Try to load the pre-made layouts
	// If it fails, we can still play the game, but players will just have randoly generated maps (not as fun)
	err = internal.InitializePremadelayouts("./../images")
	if err != nil {
		logger.Printf("Failed to load Premade Layouts: %v", err)
	}

	sessionManager := internal.NewSessionManager(logger)
	go sessionManager.MonitorSessions()

	// Are we testing on local?
	testingLocal := os.Getenv(testEnv)
	if testingLocal != "" {
		// In test mode, the frontend is served separately by the Vite dev server,
		// so we only need to handle websocket connections here.
		http.HandleFunc("/ws", sessionManager.HandleConnection(&testUpgrader))
	} else {
		// Before running the server, check if the front end files exists
		_, err = os.Stat(frontEndDirectory)
		if os.IsNotExist(err) {
			logger.Printf("Front end %s folder isn't in directory", frontEndDirectory)
			return
		}

		// Want to host the javascript files alongside handling websocket connections
		fs := http.FileServer(http.Dir("./dist"))
		http.Handle("/", fs)

		http.HandleFunc("/ws", sessionManager.HandleConnection(&upgrader))
	}

	// wss (secure websockets) should use port 443, as the standard
	logger.Printf("WebSocket server started on :443")
	err = http.ListenAndServe(":443", nil)
	if err != nil {
		logger.Printf("Error starting server: %s", err.Error())
	}

}
