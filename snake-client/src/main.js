import { Application, Graphics, Text, TextStyle } from 'pixi.js';
import { Snake,Tuple,Entity,EntityTypes} from './snake';
import { Session, SessionState} from './session';
import * as protohandler from "./protoHandler.js"
await protohandler.InitializeProtobuf();

const gridSize = 30; // Size of each square
const numRows = 20;
const numCols = 20;
const startX = 100;
const startY = 100;

const eyeOffset = gridSize / 3;

const gameSession = new Session();

const textMap = new Map();

(async () => {
  // Create a new application
  const app = new Application();

  // Initialize the application
  await app.init({ background: 'black', resizeTo: window });

  // Append the application canvas to the document body
  document.getElementById("pixi-container").appendChild(app.canvas)
  
  const graphics = new Graphics();

  // Listen for animate update
  app.ticker.add((time) => {
    graphics.clear();

    // Handle different session states
    switch (gameSession.state) {
      case SessionState.CONNECTING:
        // Display "connecting to server" message
        const connectingText = new Text({
          text: 'Connecting to server...',
          style: new TextStyle({
            fontSize: 24,
            fill: 'white'
          })
        });
        connectingText.x = startX;
        connectingText.y = startY - 50;
        app.stage.addChild(connectingText);
        break;

      case SessionState.CONNECTED:
      case SessionState.READY:
        // Display player states using Player.State from players collection
        let yOffset = startY - 50;
        gameSession.players.forEach((player, playerID) => {
          // Map PlayerState enum to readable string
          const stateNames = ['UNKNOWN', 'CONNECTING', 'CONNECTED', 'READY', 'PLAYING', 'DONE'];
          const stateValue = player.state || 2; // Default to CONNECTED if not set
          const stringValue = stateNames[stateValue] || 'UNKNOWN';

          if(textMap.has(playerID)) {
            let textValue = textMap.get(playerID)
            textValue.text = `Player ${playerID}: ${stringValue}`
          } else {
            const stateText = new Text({
              text: `Player ${playerID}: ${stringValue}`,
              style: new TextStyle({
                fontSize: 18,
                fill: player.colour || 'white'
              })
            });
            stateText.x = startX;
            stateText.y = yOffset;
            app.stage.addChild(stateText);
            textMap.set(playerID,stateText);
          }
          yOffset += 25;
        });
        break;

      case SessionState.PLAYING:
         // Draw the grid
        for (let row = 0; row < numRows; row++) {
          for (let col = 0; col < numCols; col++) {
            const x = col * gridSize + startX;
            const y = row * gridSize + startY;
            graphics.rect(x, y, gridSize, gridSize);
            graphics.stroke({ width: 2, colour: "white" });
          }
        }

        // Draw entities from gameSession
        gameSession.entities.forEach(entity => {
          var colour = "white";
          switch (entity.type) {
            case EntityTypes.BODY:
              if (gameSession.players.has(entity.playerID)) {
                const existingPlayer = gameSession.players.get(entity.playerID);
                colour = existingPlayer.colour;
              }
              drawSquare(graphics, entity.x, entity.y, colour);
              break;
            case EntityTypes.WALL:
              drawSquare(graphics, entity.x, entity.y, colour);
              break;
            case EntityTypes.POINT:
              drawCircle(graphics, entity.x, entity.y, colour);
              break;
          }
        });

        // Draw players from gameSession
        gameSession.players.forEach(player => {
          drawFace(graphics, player);
        });
        break;
    }

    app.stage.addChild(graphics);
  });
})();

function drawSquare(graphics,col,row, colour) {
  const x = col * gridSize + startX;
  const y = row * gridSize + startY;

  // Draw the square
  graphics.rect(x, y, gridSize, gridSize);
  graphics.fill(colour);
  graphics.stroke({ width: 2, colour: colour });
}

function drawFace(graphics, player) {
  const x = player.x * gridSize + startX;
  const y = player.y * gridSize + startY;
  var eye1 = new Tuple(x,y);
  var eye2 = new Tuple(x,y);

  switch (player.direction) {
    case 1:
      // Up
      eye1.x = x + eyeOffset;
      eye1.y = y + eyeOffset;
      eye2.x = x + (eyeOffset * 2);
      eye2.y = y + eyeOffset;
      break;
    case 2:
      // Down
      eye1.x = x + eyeOffset;
      eye1.y = y + (eyeOffset * 2);
      eye2.x = x + (eyeOffset * 2);
      eye2.y = y + (eyeOffset * 2);
      break;
    case 3:
      // Left
      eye1.x = x + eyeOffset;
      eye1.y = y + eyeOffset;
      eye2.x = x + eyeOffset;
      eye2.y = y + (eyeOffset * 2);
      break;
    case 4:
      // Right
      eye1.x = x + (eyeOffset * 2);
      eye1.y = y + eyeOffset;
      eye2.x = x + (eyeOffset * 2);
      eye2.y = y + (eyeOffset * 2);
      break;
  }

  graphics.circle(eye1.x, eye1.y, 3);
  graphics.fill("white");
  graphics.stroke({ width: 2, colour: "white" });

  graphics.circle(eye2.x, eye2.y, 3);
  graphics.fill("white");
  graphics.stroke({ width: 2, colour: "white" });
}

function drawCircle(graphics,col,row, colour) {
  // Circles should be in the center of the square
  const x = ((col + 0.5) * gridSize) + startX;
  const y = ((row + 0.5) * gridSize) + startY;

  // Draw the circle
  graphics.circle(x, y, gridSize / 4);
  graphics.fill(colour);
  graphics.stroke({ width: 2, colour: colour });
}

// To handle player input
document.addEventListener('keyup', (event) => {
  switch (gameSession.state) {
    case SessionState.CONNECTED:
    case SessionState.READY:
      handlePlayerReadyUp(event)
      break;
    case SessionState.PLAYING:
      handlePlayerInput(event)
      break;
  }
});

function handlePlayerReadyUp(event) {
  if (event.key == "Enter") {
    protohandler.SendReadyUpMessage(gameSession.playerEntity.id, gameSession.socket)
    gameSession.state = SessionState.READY;
  }
}

function handlePlayerInput(event) {
  let direction = 0
    switch (event.code) {
      case "KeyW":
        direction = 1;
        break;
      case "KeyS":
        direction = 2;
        break;
      case "KeyA":
        direction = 3;
        break;
      case "KeyD":
        direction = 4;
        break;
    }
    protohandler.SendInputMessage(gameSession.playerEntity.id, direction, gameSession.socket)
}