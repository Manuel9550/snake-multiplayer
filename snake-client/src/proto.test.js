import { expect, test } from "vitest";
import * as protohandler from "./protoHandler.js";

await protohandler.InitializeProtobuf();

test("successfully parse a proto message", () => {
  var payload = { PlayerID: "AwesomeString" };
  var errMsg = protohandler.ProtoPlayer.verify(payload);
  expect(errMsg).toBe(null);

  var buffer = protohandler.ProtoPlayer.encode(payload).finish();
  var message = protohandler.ProtoPlayer.decode(buffer);
  console.log(message);
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_WELCOME with valid data", () => {
  const playerPayload = {
    PlayerID: "player123",
    Body: [
      { X: 10, Y: 20 },
      { X: 10, Y: 21 },
      { X: 10, Y: 22 },
    ],
    PlayerDirection: 1, // DIRECTION_UP
    Head: { X: 10, Y: 20 },
  };

  const envelopePayload = {
    Type: 1, // MESSAGE_TYPE_WELCOME
    Welcome: playerPayload,
  };

  const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const result = protohandler.DecodeAndValidateEnvelope(buffer);

  expect(result.isValid).toBe(true);
  expect(result.type).toBe("MESSAGE_TYPE_WELCOME");
  expect(result.data.PlayerID).toBe("player123");
  expect(result.data.Body.length).toBe(3);
  expect(result.error).toBeUndefined();
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_WELCOME with ArrayBuffer input", () => {
  const playerPayload = {
    PlayerID: "player456",
    Body: [{ X: 5, Y: 5 }],
    PlayerDirection: 2, // DIRECTION_DOWN
    Head: { X: 5, Y: 5 },
  };

  const envelopePayload = {
    Type: 1, // MESSAGE_TYPE_WELCOME
    Welcome: playerPayload,
  };

  const uint8Buffer =
    protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const arrayBuffer = uint8Buffer.buffer.slice(
    uint8Buffer.byteOffset,
    uint8Buffer.byteOffset + uint8Buffer.byteLength,
  );

  const result = protohandler.DecodeAndValidateEnvelope(arrayBuffer);

  expect(result.isValid).toBe(true);
  expect(result.type).toBe("MESSAGE_TYPE_WELCOME");
  expect(result.data.PlayerID).toBe("player456");
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_SESSION_DATA with multiple players and entities", () => {
  const gameDataPayload = {
    PlayerData: [
      {
        PlayerID: "player1",
        Body: [{ X: 1, Y: 1 }],
        PlayerDirection: 3, // DIRECTION_LEFT
        Head: { X: 1, Y: 1 },
      },
      {
        PlayerID: "player2",
        Body: [{ X: 10, Y: 10 }],
        PlayerDirection: 4, // DIRECTION_RIGHT
        Head: { X: 10, Y: 10 },
      },
    ],
    EntityData: [
      {
        ID: "testID",
        PlayerID: "entity1",
        Body: { X: 5, Y: 5 },
        Type: 2, // ENTITY_TYPE_POINT
      },
      {
        ID: "testID2",
        PlayerID: "entity2",
        Body: { X: 15, Y: 15 },
        Type: 1, // ENTITY_TYPE_WALL
      },
    ],
  };

  const envelopePayload = {
    Type: 2, // MESSAGE_TYPE_SESSION_DATA
    SessionData: {
      SessionID: "test-session",
      State: 3, // SESSION_STATE_RUNNING
      GameData: gameDataPayload,
    },
  };

  const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const result = protohandler.DecodeAndValidateEnvelope(buffer);

  expect(result.isValid).toBe(true);
  expect(result.type).toBe("MESSAGE_TYPE_SESSION_DATA");
  expect(result.data.GameData.PlayerData.length).toBe(2);
  expect(result.data.GameData.PlayerData[0].PlayerID).toBe("player1");
  expect(result.data.GameData.PlayerData[1].PlayerID).toBe("player2");
  expect(result.data.GameData.EntityData.length).toBe(2);
  expect(result.data.GameData.EntityData[0].PlayerID).toBe("entity1");
  expect(result.data.GameData.EntityData[0].ID).toBe("testID");
  expect(result.data.GameData.EntityData[1].PlayerID).toBe("entity2");
  expect(result.data.GameData.EntityData[1].ID).toBe("testID2");
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_UNKNOWN", () => {
  const envelopePayload = {
    Type: 0, // MESSAGE_TYPE_UNKNOWN
  };

  const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const result = protohandler.DecodeAndValidateEnvelope(buffer);

  expect(result.isValid).toBe(true);
  expect(result.type).toBe("MESSAGE_TYPE_UNKNOWN");
  expect(result.warning).toBe("Unknown message type");
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_WELCOME missing Welcome field", () => {
  const envelopePayload = {
    Type: 1, // MESSAGE_TYPE_WELCOME but no Welcome field
  };

  const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const result = protohandler.DecodeAndValidateEnvelope(buffer);

  expect(result.isValid).toBe(false);
  expect(result.error).toContain("Welcome field is missing");
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_SESSION_DATA missing SessionData field", () => {
  const envelopePayload = {
    Type: 2, // MESSAGE_TYPE_SESSION_DATA but no SessionData field
  };

  const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const result = protohandler.DecodeAndValidateEnvelope(buffer);

  expect(result.isValid).toBe(false);
  expect(result.error).toContain("SessionData field is missing");
});

test("DecodeAndValidateEnvelope - invalid binary data format", () => {
  const invalidData = "not binary data";
  const result = protohandler.DecodeAndValidateEnvelope(invalidData);

  expect(result.isValid).toBe(false);
  expect(result.error).toContain("Invalid binary data format");
});

test("DecodeAndValidateEnvelope - corrupted binary data", () => {
  const corruptedData = new Uint8Array([0xff, 0xff, 0xff, 0xff]);
  const result = protohandler.DecodeAndValidateEnvelope(corruptedData);

  expect(result.isValid).toBe(false);
  expect(result.error).toBeDefined();
});

test("DecodeAndValidateEnvelope - MESSAGE_TYPE_SESSION_DATA with empty arrays", () => {
  const gameDataPayload = {
    PlayerData: [],
    EntityData: [],
  };

  const envelopePayload = {
    Type: 2, // MESSAGE_TYPE_SESSION_DATA
    SessionData: {
      SessionID: "empty-session",
      State: 2, // SESSION_STATE_SETUP
      GameData: gameDataPayload,
    },
  };

  const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
  const result = protohandler.DecodeAndValidateEnvelope(buffer);

  expect(result.isValid).toBe(true);
  expect(result.type).toBe("MESSAGE_TYPE_SESSION_DATA");
  expect(result.data.GameData.PlayerData.length).toBe(0);
  expect(result.data.GameData.EntityData.length).toBe(0);
});
