var protobuf = require("protobufjs");
export var ProtoEnvelope;
export var ProtoInput;
export var ProtoDirectionTypes;
export var ProtoPlayer;
export var ProtoGameData;
export var ProtoEntity;
export var ProtoBodyField;
export var ProtoEntityTypes;
export var ProtoWelcome;
export var ProtoPlayerStates;
export var ProtoPlayerState;
export var ProtoPlayerStateEnum;
export var ProtoServerStateMessage;
let messageTypes = []

// Load protobuf schema
export async function InitializeProtobuf() {
  try {
    const root = await protobuf.load('./src/proto/player.proto');
    ProtoEnvelope = root.lookupType('Envelope');
    console.log("Protobuf Envelope loaded successfully");

    ProtoPlayer = root.lookupType('Player');
    console.log("Protobuf Player loaded successfully");

    ProtoGameData = root.lookupType('GameData');
    console.log("Protobuf GameData loaded successfully");

    ProtoWelcome = root.lookupType('Welcome');
    console.log("Protobuf Welcome loaded successfully");

    ProtoPlayerStates = root.lookupType('PlayerStates');
    console.log("Protobuf PlayerStates loaded successfully");

    const rootInputs = await protobuf.load('./src/proto/inputs.proto');
    ProtoInput = rootInputs.lookupType('Input');
    console.log("Protobuf Input loaded successfully");

    ProtoDirectionTypes = rootInputs.lookupEnum("Direction");
    console.log("Protobuf Direction loaded successfully");

    ProtoPlayerState = rootInputs.lookupType("PlayerState");
    console.log("Protobuf PlayerState loaded successfully");

    ProtoPlayerStateEnum = rootInputs.lookupEnum("State");
    console.log("Protobuf State Enum loaded successfully");

    const rootEntity = await protobuf.load('./src/proto/entity.proto');
    ProtoEntity = rootEntity.lookupType('Entity');
    console.log("Protobuf Entity loaded successfully");

    ProtoBodyField = rootEntity.lookupType('BodyField');
    console.log("Protobuf BodyField loaded successfully");

    ProtoEntityTypes = rootEntity.lookupEnum("EntityType");
    console.log("Protobuf EntityType loaded successfully");

    const rootSession = await protobuf.load('./src/proto/server.proto');
    ProtoServerStateMessage = rootSession.lookupType('ServerStateMessage');
    console.log("ProtoServerStateMessage Entity loaded successfully");

    messageTypes.push(ProtoEnvelope, ProtoInput, ProtoGameData, ProtoEntity,ProtoServerStateMessage)
  } catch (error) {
    console.error("Failed to load protobuf schema:", error);
  }
}




export function HandleEntityMessage(messageData) {
  try {
    // Check if the message is binary data (ArrayBuffer or Uint8Array)
    let binaryData;
    if (messageData instanceof ArrayBuffer) {
      binaryData = new Uint8Array(messageData);
    } else if (messageData instanceof Uint8Array) {
      binaryData = messageData;
    } else if (typeof messageData === 'string') {
      // If it's a string, it might be a text message - handle accordingly
      console.log("Received text message:", messageData);
      return;
    } else {
      console.error("Unknown message format:", typeof messageData);
      return;
    }

    // Deserialize the protobuf message
    return ProtoEnvelope.decode(binaryData);
    
  } catch (error) {
    console.error("Error parsing protobuf message:", error);
  }
}

export function DecodeAndValidateEnvelope(binaryData) {
  try {
    // Convert input to Uint8Array if necessary
    let data;
    if (binaryData instanceof ArrayBuffer) {
      data = new Uint8Array(binaryData);
    } else if (binaryData instanceof Uint8Array) {
      data = binaryData;
    } else {
      throw new Error("Invalid binary data format. Expected ArrayBuffer or Uint8Array.");
    }

    // Decode the Envelope message
    const envelope = ProtoEnvelope.decode(data);

    // Validate the decoded message
    const validationError = ProtoEnvelope.verify(envelope);
    if (validationError) {
      throw new Error(`Envelope validation failed: ${validationError}`);
    }

    // Validate based on Type field
    switch (envelope.Type) {
      case 0: // MESSAGE_TYPE_UNKNOWN
        console.warn("Received envelope with MESSAGE_TYPE_UNKNOWN");
        return {
          isValid: true,
          type: "MESSAGE_TYPE_UNKNOWN",
          envelope: envelope,
          warning: "Unknown message type"
        };

      case 1: // MESSAGE_TYPE_WELCOME
        if (!envelope.PlayerData) {
          throw new Error("MESSAGE_TYPE_WELCOME expected but PlayerData field is missing");
        }
        // Validate the Welcome message
        const welcomeError = ProtoWelcome.verify(envelope.PlayerData);
        if (welcomeError) {
          throw new Error(`Welcome validation failed: ${welcomeError}`);
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_WELCOME",
          envelope: envelope,
          data: envelope.PlayerData
        };

      case 2: // MESSAGE_TYPE_PLAYER_STATES
        if (!envelope.PlayerStates) {
          throw new Error("MESSAGE_TYPE_PLAYER_STATES expected but PlayerStates field is missing");
        }
        // Validate the PlayerStates message
        const playerStatesError = ProtoPlayerStates.verify(envelope.PlayerStates);
        if (playerStatesError) {
          throw new Error(`PlayerStates validation failed: ${playerStatesError}`);
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_PLAYER_STATES",
          envelope: envelope,
          data: envelope.PlayerStates
        };
      case 3: // MESSAGE_TYPE_SERVER_STATE
        if (!envelope.ServerState) {
          throw new Error("MESSAGE_TYPE_SERVER_STATE expected but ServerState field is missing");
        }
        // Validate the SessionStates message
        const serverStatesError = ProtoServerStateMessage.verify(envelope.ServerState);
        if (serverStatesError) {
          throw new Error(`SessionStates validation failed: ${serverStatesError}`);
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_SERVER_STATE",
          envelope: envelope,
          data: envelope.ServerState
        };

      case 4: // MESSAGE_TYPE_PLAYER
        if (!envelope.Data) {
          throw new Error("MESSAGE_TYPE_PLAYER expected but Data field is missing");
        }
        // Validate the Player message
        const playerError = ProtoPlayer.verify(envelope.Data);
        if (playerError) {
          throw new Error(`Player validation failed: ${playerError}`);
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_PLAYER",
          envelope: envelope,
          data: envelope.Data
        };

      case 5: // MESSAGE_TYPE_GAME_DATA
        if (!envelope.GameState) {
          throw new Error("MESSAGE_TYPE_GAME_DATA expected but GameState field is missing");
        }
        // Validate the GameData message
        const gameDataError = ProtoGameData.verify(envelope.GameState);
        if (gameDataError) {
          throw new Error(`GameData validation failed: ${gameDataError}`);
        }
        // Validate each Player in the PlayerData array
        if (envelope.GameState.PlayerData) {
          for (let i = 0; i < envelope.GameState.PlayerData.length; i++) {
            const playerValidationError = ProtoPlayer.verify(envelope.GameState.PlayerData[i]);
            if (playerValidationError) {
              throw new Error(`Player validation failed at index ${i}: ${playerValidationError}`);
            }
          }
        }
        // Validate each Entity in the EntityData array
        if (envelope.GameState.EntityData) {
          for (let i = 0; i < envelope.GameState.EntityData.length; i++) {
            const entityError = ProtoEntity.verify(envelope.GameState.EntityData[i]);
            if (entityError) {
              throw new Error(`Entity validation failed at index ${i}: ${entityError}`);
            }
          }
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_GAME_DATA",
          envelope: envelope,
          data: envelope.GameState
        };

      default:
        throw new Error(`Unknown MessageType: ${envelope.Type}`);
    }

  } catch (error) {
    console.error("Error decoding/validating envelope:", error.message);
    return {
      isValid: false,
      error: error.message
    };
  }
}

export function SendInputMessage(id, type, socket) {
    const payload = {
        PlayerID: id,
        InputType: type,
    };

    try {
      const errMsg = ProtoInput.verify(payload);
      if (errMsg) throw Error(errMsg);
      const message = ProtoInput.create(payload);
      // Encode the message to a buffer
      const buffer = ProtoInput.encode(message).finish();
      socket.send(buffer)
    } catch (error) {
      console.error("Error sending player input envelope:", error.message);
    }
}

export function SendStateMessage(player,state,socket) {
    console.log(state)
    const payload = {
        PlayerID: player.id,
        State: state,
        Colour: player.colour,
    };

    console.log(payload)

    try {
      const errMsg = ProtoPlayerState.verify(payload);
      if (errMsg) throw Error(errMsg);
      const message = ProtoPlayerState.create(payload);
      // Encode the message to a buffer
      const buffer = ProtoPlayerState.encode(message).finish();
      socket.send(buffer)
    } catch (error) {
      console.error("Error sending player input envelope:", error.message);
    }
}



