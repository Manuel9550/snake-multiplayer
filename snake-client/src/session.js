import * as protohandler from "./protoHandler.js"
import { Snake,Tuple,Entity,EntityTypes, PlayerState} from './snake';

const connectionString = "ws://localhost:8080/ws";

// enum State {
//     STATE_UNKNOWN = 0;
//     STATE_CONNECTING = 1;
//     STATE_CONNECTED = 2;
//     STATE_READY = 3;
//     STATE_PLAYING = 4;
//     STATE_DONE = 5;
// }

export const SessionState = Object.freeze({
    UNKNOWN: 0,
    CONNECTING: 1,
    CONNECTED: 2,
    READY: 3,
    PLAYING: 4,
    DONE: 5
});


export class Session {
    state = SessionState.CONNECTING;
    socket = null;
    playerEntity = null;
    players = new Map();
    playerStates = new Map();
    entities = new Map();

    

    constructor() {
            // Create WebSocket connection.
            this.socket = new WebSocket(connectionString);

            // Set binary type to handle protobuf messages
            this.socket.binaryType = 'arraybuffer';

            // Connection opened
            this.socket.addEventListener("open", (event) => {
                console.log("WebSocket connected");
                this.socket.send("Hello Server");
            });

            this.socket.addEventListener("message",this.handleProtoMessage.bind(this))
    }

    handleProtoMessage(event) {
        const result = protohandler.DecodeAndValidateEnvelope(event.data);
        
        if (!result.isValid) {
            console.error("Failed to decode envelope:", result.error);
            return;
        }
        
        // Depending on the state, we only care about certain message 
        // type
        switch (this.state) {
            case SessionState.CONNECTING:
                // Only care about Welcome and PlayerStates messages
                if (result.type === "MESSAGE_TYPE_WELCOME") {
                    const playerData = result.data;
                    this.newPlayer(playerData);
                }
                if (result.type === "MESSAGE_TYPE_PLAYER_STATES") {
                    const playerStates = result.data;
                    this.updatePlayerStates(playerStates);
                }
                break;
            case SessionState.CONNECTED:
                if (result.type === "MESSAGE_TYPE_PLAYER_STATES") {
                    const playerStates = result.data;
                    this.updatePlayerStates(playerStates);
                }
                if (result.type === "MESSAGE_TYPE_SERVER_STATE") {
                    const serverState = result.data;
                    this.updateServerState(serverState);
                }
                break;
            case SessionState.PLAYING:
                if (result.type === "MESSAGE_TYPE_GAME_DATA") {
                    const playersData = result.data.PlayerData;
                    this.updateGameData(playersData);
                }
                break;
            }
    }

    newPlayer(playerData) {
        this.playerEntity = new Snake(playerData.PlayerID, 0, 0, playerData.Colour);
        this.players.set(playerData.PlayerID, this.playerEntity);
        this.state = SessionState.CONNECTED;
        console.log("New player: ");
        console.log(this.playerEntity);
    }

    updatePlayerStates(statesData) {
        console.log(statesData)
        const stateArray = Array.from(statesData.PlayerStates) 
        stateArray.forEach(playerState => {
            if (this.playerStates.has(playerState.PlayerID)) {
                console.log("Dont have " + playerState.PlayerID);
                // Update existing player state
                const existingPlayerState = this.playerStates.get(playerState.PlayerID);
                // The playerID and colour shouldn't change, just update the state
                existingPlayerState.state = playerState.State;
            } else {
                console.log("Have " + playerState.PlayerID);
                // Create new player state only if it doesn't exist
                const newPlayerState = new PlayerState(playerState.PlayerID, playerState.State, playerState.Colour);
                this.playerStates.set(newPlayerState.playerID, newPlayerState);
                console.log(this.playerStates)
            }
        });
    }

    updateServerState(serverState) {
       switch(serverState) {
        case 3:
            this.state = PLAYING
            break;
        case 4:
            this.state = DONE
            break;
        }
    }

    updateGameData(playersData) {
        playersData.forEach(playerData => {
            if (this.players.has(playerData.PlayerID)) {
                // Update existing player
                const existingPlayer = this.players.get(playerData.PlayerID);
                existingPlayer.direction = playerData.PlayerDirection;
                existingPlayer.x = playerData.Head.X;
                existingPlayer.y = playerData.Head.Y;
                existingPlayer.body = playerData.Body.map(bodyField => new Tuple(bodyField.X, bodyField.Y));
            } else {
                // Create new player only if it doesn't exist
                const newPlayer = new Snake(playerData.PlayerID, playerData.Head.X, playerData.Head.Y, playerData.Colour);
                newPlayer.direction = playerData.PlayerDirection;
                newPlayer.body = playerData.Body.map(bodyField => new Tuple(bodyField.X, bodyField.Y));
                this.players.set(playerData.PlayerID, newPlayer);
                console.log(newPlayer);
            }
        });
        
        const entitiesData = result.data.EntityData;
        // The server sends all active entities: we may need to prune entities that are no longer active
        let activeMap = new Map()
        console.log("server sent " + entitiesData.length + " entities")
        console.log(entitiesData)
        entitiesData.forEach(entityData => {
            activeMap.set(entityData.ID,true);
            if (entities.has(entityData.ID)) {
                // Update existing entity
                const existingEntity = this.entities.get(entityData.ID);
                existingEntity.x = entityData.Body.X;
                existingEntity.y = entityData.Body.Y;
                existingEntity.type = entityData.Type;
            } else {
                // Create new entity if it doesn't exist
                const newEntity = new Entity(entityData.ID, entityData.PlayerID, entityData.Body.X,entityData.Body.Y, entityData.Type);
                this.entities.set(newEntity.id, newEntity);
            }
        });
        
        this.entities.forEach(currentEntityInMap => {
            if (!activeMap.has(currentEntityInMap.id)) {
            this.entities.delete(currentEntityInMap.id)
            }
        });
    }
}