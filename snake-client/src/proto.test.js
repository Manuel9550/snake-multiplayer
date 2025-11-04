import { expect, test } from 'vitest'
import * as protohandler from "./protoHandler.js"

await protohandler.InitializeProtobuf();

test('successfully parse a proto message', () => {
    var payload = { "PlayerID": "AwesomeString" };
    var errMsg = protohandler.ProtoPlayer.verify(payload)
    expect(errMsg).toBe(null)

    var buffer = protohandler.ProtoPlayer.encode(payload).finish();
    var message = protohandler.ProtoPlayer.decode(buffer);
    console.log(message)

})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_PLAYER with valid data', () => {
    const playerPayload = {
        PlayerID: "player123",
        Body: [
            { X: 10, Y: 20 },
            { X: 10, Y: 21 },
            { X: 10, Y: 22 }
        ],
        PlayerDirection: 1, // DIRECTION_UP
        Head: { X: 10, Y: 20 }
    };

    const envelopePayload = {
        Type: 1, // MESSAGE_TYPE_PLAYER
        Data: playerPayload
    };

    const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const result = protohandler.DecodeAndValidateEnvelope(buffer);

    expect(result.isValid).toBe(true);
    expect(result.type).toBe("MESSAGE_TYPE_PLAYER");
    expect(result.data.PlayerID).toBe("player123");
    expect(result.data.Body.length).toBe(3);
    expect(result.error).toBeUndefined();
})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_PLAYER with ArrayBuffer input', () => {
    const playerPayload = {
        PlayerID: "player456",
        Body: [{ X: 5, Y: 5 }],
        PlayerDirection: 2, // DIRECTION_DOWN
        Head: { X: 5, Y: 5 }
    };

    const envelopePayload = {
        Type: 1, // MESSAGE_TYPE_PLAYER
        Data: playerPayload
    };

    const uint8Buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const arrayBuffer = uint8Buffer.buffer.slice(uint8Buffer.byteOffset, uint8Buffer.byteOffset + uint8Buffer.byteLength);

    const result = protohandler.DecodeAndValidateEnvelope(arrayBuffer);

    expect(result.isValid).toBe(true);
    expect(result.type).toBe("MESSAGE_TYPE_PLAYER");
    expect(result.data.PlayerID).toBe("player456");
})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_GAME_DATA with multiple players and entities', () => {
    const gameDataPayload = {
        PlayerData: [
            {
                PlayerID: "player1",
                Body: [{ X: 1, Y: 1 }],
                PlayerDirection: 3, // DIRECTION_LEFT
                Head: { X: 1, Y: 1 }
            },
            {
                PlayerID: "player2",
                Body: [{ X: 10, Y: 10 }],
                PlayerDirection: 4, // DIRECTION_RIGHT
                Head: { X: 10, Y: 10 }
            }
        ],
        EntityData: [
            {
                ID: "testID",
                PlayerID: "entity1",
                Body: { X: 5, Y: 5 },
                Type: 2 // ENTITY_TYPE_POINT
            },
            {
                ID: "testID2",
                PlayerID: "entity2",
                Body: { X: 15, Y: 15 },
                Type: 1 // ENTITY_TYPE_WALL
            }
        ]
    };

    const envelopePayload = {
        Type: 2, // MESSAGE_TYPE_GAME_DATA
        GameState: gameDataPayload
    };

    const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const result = protohandler.DecodeAndValidateEnvelope(buffer);

    expect(result.isValid).toBe(true);
    expect(result.type).toBe("MESSAGE_TYPE_GAME_DATA");
    expect(result.data.PlayerData.length).toBe(2);
    expect(result.data.PlayerData[0].PlayerID).toBe("player1");
    expect(result.data.PlayerData[1].PlayerID).toBe("player2");
    expect(result.data.EntityData.length).toBe(2);
    expect(result.data.EntityData[0].PlayerID).toBe("entity1");
    expect(result.data.EntityData[0].ID).toBe("testID");
    expect(result.data.EntityData[1].PlayerID).toBe("entity2");
    expect(result.data.EntityData[1].ID).toBe("testID2");
})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_UNKNOWN', () => {
    const envelopePayload = {
        Type: 0 // MESSAGE_TYPE_UNKNOWN
    };

    const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const result = protohandler.DecodeAndValidateEnvelope(buffer);

    expect(result.isValid).toBe(true);
    expect(result.type).toBe("MESSAGE_TYPE_UNKNOWN");
    expect(result.warning).toBe("Unknown message type");
})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_PLAYER missing Data field', () => {
    const envelopePayload = {
        Type: 1 // MESSAGE_TYPE_PLAYER but no Data field
    };

    const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const result = protohandler.DecodeAndValidateEnvelope(buffer);

    expect(result.isValid).toBe(false);
    expect(result.error).toContain("Data field is missing");
})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_GAME_DATA missing GameState field', () => {
    const envelopePayload = {
        Type: 2 // MESSAGE_TYPE_GAME_DATA but no GameState field
    };

    const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const result = protohandler.DecodeAndValidateEnvelope(buffer);

    expect(result.isValid).toBe(false);
    expect(result.error).toContain("GameState field is missing");
})

test('DecodeAndValidateEnvelope - invalid binary data format', () => {
    const invalidData = "not binary data";
    const result = protohandler.DecodeAndValidateEnvelope(invalidData);

    expect(result.isValid).toBe(false);
    expect(result.error).toContain("Invalid binary data format");
})

test('DecodeAndValidateEnvelope - corrupted binary data', () => {
    const corruptedData = new Uint8Array([0xFF, 0xFF, 0xFF, 0xFF]);
    const result = protohandler.DecodeAndValidateEnvelope(corruptedData);

    expect(result.isValid).toBe(false);
    expect(result.error).toBeDefined();
})

test('DecodeAndValidateEnvelope - MESSAGE_TYPE_GAME_DATA with empty arrays', () => {
    const gameDataPayload = {
        PlayerData: [],
        EntityData: []
    };

    const envelopePayload = {
        Type: 2, // MESSAGE_TYPE_GAME_DATA
        GameState: gameDataPayload
    };

    const buffer = protohandler.ProtoEnvelope.encode(envelopePayload).finish();
    const result = protohandler.DecodeAndValidateEnvelope(buffer);

    expect(result.isValid).toBe(true);
    expect(result.type).toBe("MESSAGE_TYPE_GAME_DATA");
    expect(result.data.PlayerData.length).toBe(0);
    expect(result.data.EntityData.length).toBe(0);
})