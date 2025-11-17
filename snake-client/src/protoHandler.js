var protobuf = require("protobufjs");
export var ProtoEnvelope;
export var ProtoInput;
export var ProtoReadyUp;
export var ProtoDirectionTypes;
export var ProtoPlayer;
export var ProtoGameData;
export var ProtoEntity;
export var ProtoBodyField;
export var ProtoEntityTypes;
export var ProtoPlayerStateEnum;
export var ProtoSessionData;
export var ProtoSessionStateEnum;
export var ProtoMessageTypeEnum;
let messageTypes = [];

// Load protobuf schema
export async function InitializeProtobuf() {
  try {
    // Load session.proto which imports game.proto
    const root = await protobuf.load("./src/proto/session.proto");
    ProtoEnvelope = root.lookupType("Envelope");
    console.log("Protobuf Envelope loaded successfully");

    ProtoSessionData = root.lookupType("SessionData");
    console.log("Protobuf SessionData loaded successfully");

    ProtoPlayer = root.lookupType("Player");
    console.log("Protobuf Player loaded successfully");

    ProtoGameData = root.lookupType("GameData");
    console.log("Protobuf GameData loaded successfully");

    ProtoEntity = root.lookupType("Entity");
    console.log("Protobuf Entity loaded successfully");

    ProtoBodyField = root.lookupType("BodyField");
    console.log("Protobuf BodyField loaded successfully");

    ProtoEntityTypes = root.lookupEnum("EntityType");
    console.log("Protobuf EntityType loaded successfully");

    ProtoPlayerStateEnum = root.lookupEnum("PlayerState");
    console.log("Protobuf PlayerState Enum loaded successfully");

    ProtoSessionStateEnum = root.lookupEnum("SessionState");
    console.log("Protobuf SessionState Enum loaded successfully");

    ProtoMessageTypeEnum = root.lookupEnum("MessageType");
    console.log("Protobuf MessageType Enum loaded successfully");

    // Load inputs.proto
    const rootInputs = await protobuf.load("./src/proto/inputs.proto");
    ProtoInput = rootInputs.lookupType("Input");
    console.log("Protobuf Input loaded successfully");

    ProtoReadyUp = rootInputs.lookupType("ReadyUp");
    console.log("Protobuf ReadyUp loaded successfully");

    ProtoDirectionTypes = rootInputs.lookupEnum("Direction");
    console.log("Protobuf Direction loaded successfully");

    messageTypes.push(
      ProtoEnvelope,
      ProtoInput,
      ProtoGameData,
      ProtoEntity,
      ProtoSessionData,
    );
  } catch (error) {
    console.error("Failed to load protobuf schema:", error);
  }
}

await InitializeProtobuf();

export function HandleEntityMessage(messageData) {
  try {
    // Check if the message is binary data (ArrayBuffer or Uint8Array)
    let binaryData;
    if (messageData instanceof ArrayBuffer) {
      binaryData = new Uint8Array(messageData);
    } else if (messageData instanceof Uint8Array) {
      binaryData = messageData;
    } else if (typeof messageData === "string") {
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
      throw new Error(
        "Invalid binary data format. Expected ArrayBuffer or Uint8Array.",
      );
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
          warning: "Unknown message type",
        };

      case 1: {
        // MESSAGE_TYPE_WELCOME
        if (!envelope.Welcome) {
          throw new Error(
            "MESSAGE_TYPE_WELCOME expected but Welcome field is missing",
          );
        }
        // Validate the Welcome message (which is a Player type)
        const welcomeError = ProtoPlayer.verify(envelope.Welcome);
        if (welcomeError) {
          throw new Error(`Welcome validation failed: ${welcomeError}`);
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_WELCOME",
          envelope: envelope,
          data: envelope.Welcome,
        };
      }
      case 2: {
        // MESSAGE_TYPE_SESSION_DATA
        if (!envelope.SessionData) {
          throw new Error(
            "MESSAGE_TYPE_SESSION_DATA expected but SessionData field is missing",
          );
        }
        // Validate the SessionData message
        const sessionDataError = ProtoSessionData.verify(envelope.SessionData);
        if (sessionDataError) {
          throw new Error(`SessionData validation failed: ${sessionDataError}`);
        }
        // Validate the GameData inside SessionData if present
        if (envelope.SessionData.GameData) {
          const gameDataError = ProtoGameData.verify(
            envelope.SessionData.GameData,
          );
          if (gameDataError) {
            throw new Error(`GameData validation failed: ${gameDataError}`);
          }
          // Validate each Player in the PlayerData array
          if (envelope.SessionData.GameData.PlayerData) {
            for (
              let i = 0;
              i < envelope.SessionData.GameData.PlayerData.length;
              i++
            ) {
              const playerValidationError = ProtoPlayer.verify(
                envelope.SessionData.GameData.PlayerData[i],
              );
              if (playerValidationError) {
                throw new Error(
                  `Player validation failed at index ${i}: ${playerValidationError}`,
                );
              }
            }
          }
          // Validate each Entity in the EntityData array
          if (envelope.SessionData.GameData.EntityData) {
            for (
              let i = 0;
              i < envelope.SessionData.GameData.EntityData.length;
              i++
            ) {
              const entityError = ProtoEntity.verify(
                envelope.SessionData.GameData.EntityData[i],
              );
              if (entityError) {
                throw new Error(
                  `Entity validation failed at index ${i}: ${entityError}`,
                );
              }
            }
          }
        }
        return {
          isValid: true,
          type: "MESSAGE_TYPE_SESSION_DATA",
          envelope: envelope,
          data: envelope.SessionData,
        };
      }
      default:
        throw new Error(`Unknown MessageType: ${envelope.Type}`);
    }
  } catch (error) {
    console.error("Error decoding/validating envelope:", error.message);
    return {
      isValid: false,
      error: error.message,
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
    socket.send(buffer);
  } catch (error) {
    console.error("Error sending player input envelope:", error.message);
  }
}

export function SendReadyUpMessage(playerID, socket) {
  const payload = {
    PlayerID: playerID,
  };

  console.log("Sending ready up for player:", playerID);

  try {
    const errMsg = ProtoReadyUp.verify(payload);
    if (errMsg) throw Error(errMsg);
    const message = ProtoReadyUp.create(payload);
    // Encode the message to a buffer
    const buffer = ProtoReadyUp.encode(message).finish();
    socket.send(buffer);
  } catch (error) {
    console.error("Error sending ready up message:", error.message);
  }
}
