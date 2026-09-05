import * as protohandler from "./protoHandler.js";
import { Snake, Tuple, Entity } from "./snake";

// Note: make sure it's wss (web socket secure) to avoid mixed content errors when the user laods the page with https!
// import.meta.env.DEV is set by Vite: true when running "npm run dev", false for "npm run build"
const connectionString = import.meta.env.DEV
  ? "ws://localhost:443/ws"
  : `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws`;
export const SessionState = Object.freeze({
  UNKNOWN: 0,
  CONNECTING: 1,
  CONNECTED: 2,
  READY: 3,
  PLAYING: 4,
  DONE: 5,
});

export class Session {
  state = SessionState.CONNECTING;
  socket = null;
  playerEntity = null;
  players = new Map();
  entities = new Map();
  winningPlayerID = null;

  constructor() {
    // Create WebSocket connection.
    this.socket = new WebSocket(connectionString);

    // Set binary type to handle protobuf messages
    this.socket.binaryType = "arraybuffer";

    // Connection opened
    this.socket.addEventListener("open", () => {
      this.socket.send("Hello Server");
    });

    this.socket.addEventListener("message", this.handleProtoMessage.bind(this));
  }

  handleProtoMessage(event) {
    const result = protohandler.DecodeAndValidateEnvelope(event.data);

    if (!result.isValid) {
      console.error("Failed to decode envelope:", result.error);
      return;
    }

    // Handle Welcome message - sent when player first connects
    if (result.type === "MESSAGE_TYPE_WELCOME") {
      const playerData = result.data;
      this.newPlayer(playerData);
      return;
    }

    // Handle SessionData message - contains session state and game data
    if (result.type === "MESSAGE_TYPE_SESSION_DATA") {
      const sessionData = result.data;
      this.updateSessionData(sessionData);
      return;
    }
  }

  newPlayer(playerData) {
    this.playerEntity = new Snake(playerData.PlayerID, 0, 0, playerData.Colour);
    this.players.set(playerData.PlayerID, this.playerEntity);
    this.state = SessionState.CONNECTED;
  }

  updateSessionData(sessionData) {
    // Update session state based on server's SessionState
    // Map server SessionState to client SessionState
    switch (sessionData.State) {
      case 1: // SESSION_STATE_INITIALIZING
        this.state = SessionState.CONNECTING;
        break;
      case 2: // SESSION_STATE_SETUP
        // Check if we're already in READY state, otherwise set to CONNECTED
        if (this.state !== SessionState.READY) {
          this.state = SessionState.CONNECTED;
        }
        break;
      case 3: // SESSION_STATE_RUNNING
        this.state = SessionState.PLAYING;
        break;
      case 4: // SESSION_STATE_DONE
        this.state = SessionState.DONE;
        if (sessionData.WinningPlayer) {
          this.winningPlayerID = sessionData.WinningPlayer;
        }
        break;
      default:
        console.warn("Unknown session state:", sessionData.State);
    }

    // Update game data if present
    if (sessionData.GameData) {
      this.updateGameData(sessionData.GameData);
    }
  }

  updateGameData(gameData) {
    // Similar to entities, some players could exist client side that no longer exist on the server side
    // I.E. A player disconnects in the setup phase.
    // We need to account for this and prune older players
    let activePlayerMap = new Map();
    gameData.PlayerData.forEach((playerData) => {
      // This player has data in the latest session update: it is active
      activePlayerMap.set(playerData.PlayerID, true);
      if (this.players.has(playerData.PlayerID)) {
        // Update existing player
        const existingPlayer = this.players.get(playerData.PlayerID);
        existingPlayer.direction = playerData.PlayerDirection;
        existingPlayer.x = playerData.Head.X;
        existingPlayer.y = playerData.Head.Y;
        existingPlayer.body = playerData.Body.map(
          (bodyField) => new Tuple(bodyField.X, bodyField.Y),
        );
        existingPlayer.state = playerData.State || 0;
      } else {
        // Create new player only if it doesn't exist
        const newPlayer = new Snake(
          playerData.PlayerID,
          playerData.Head.X,
          playerData.Head.Y,
          playerData.Colour,
        );
        newPlayer.direction = playerData.PlayerDirection;
        newPlayer.body = playerData.Body.map(
          (bodyField) => new Tuple(bodyField.X, bodyField.Y),
        );
        newPlayer.state = playerData.State || 0;
        this.players.set(playerData.PlayerID, newPlayer);
      }
    });

    this.players.forEach((currentPlayerInMap) => {
      if (!activePlayerMap.has(currentPlayerInMap.id)) {
        this.players.delete(currentPlayerInMap.id);
      }
    });

    const entitiesData = gameData.EntityData;
    // The server sends all active entities: we may need to prune entities that are no longer active
    let activeMap = new Map();
    entitiesData.forEach((entityData) => {
      activeMap.set(entityData.ID, true);
      if (this.entities.has(entityData.ID)) {
        // Update existing entity
        const existingEntity = this.entities.get(entityData.ID);
        existingEntity.x = entityData.Body.X;
        existingEntity.y = entityData.Body.Y;
        existingEntity.type = entityData.Type;
      } else {
        // Create new entity if it doesn't exist
        const newEntity = new Entity(
          entityData.ID,
          entityData.PlayerID,
          entityData.Body.X,
          entityData.Body.Y,
          entityData.Type,
        );
        this.entities.set(newEntity.id, newEntity);
      }
    });

    this.entities.forEach((currentEntityInMap) => {
      if (!activeMap.has(currentEntityInMap.id)) {
        this.entities.delete(currentEntityInMap.id);
      }
    });
  }
}
