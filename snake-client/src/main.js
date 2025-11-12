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
var removeTextState = 0;

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
        textMap.set("connecting",connectingText);
        break;

      case SessionState.CONNECTED:
      case SessionState.READY:
        removeText(gameSession.state)
        // Display "Hit Enter to ready up" instruction message
        if (!textMap.has("instructions")) {
          const instructionText = new Text({
            text: 'Hit Enter to ready up',
            style: new TextStyle({
              fontSize: 20,
              fill: 'yellow',
              fontWeight: 'bold'
            })
          });
          instructionText.x = startX;
          instructionText.y = startY - 80;
          app.stage.addChild(instructionText);
          textMap.set("instructions",instructionText);
        }

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
                fill: 'white'  // Changed to white
              })
            });
            stateText.x = startX + 40;  // Offset to make room for snake icon
            stateText.y = yOffset;
            app.stage.addChild(stateText);
            textMap.set(playerID,stateText);
          }

          // Draw small snake icon beside player status
          const snakeIconX = startX + 10;
          const snakeIconY = yOffset + 5;
          const snakeIconSize = 15;

          // Draw snake head (small square)
          graphics.rect(snakeIconX, snakeIconY, snakeIconSize, snakeIconSize);
          graphics.fill(player.colour);
          graphics.stroke({ width: 1, color: player.colour });

          // Draw small eyes on the snake icon
          const eyeSize = 2;
          const eyeOffsetIcon = snakeIconSize / 4;
          graphics.circle(snakeIconX + eyeOffsetIcon, snakeIconY + eyeOffsetIcon, eyeSize);
          graphics.fill("white");
          graphics.circle(snakeIconX + snakeIconSize - eyeOffsetIcon, snakeIconY + eyeOffsetIcon, eyeSize);
          graphics.fill("white");

          yOffset += 25;
        });
        break;

      case SessionState.PLAYING:
        removeText(gameSession.state)
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

      case SessionState.DONE:
        removeText(gameSession.state)

        // Display "Game Over" title
        if (!textMap.has("gameOver")) {
          const gameOverText = new Text({
            text: 'GAME OVER',
            style: new TextStyle({
              fontSize: 48,
              fill: 'red',
              fontWeight: 'bold'
            })
          });
          gameOverText.x = startX + 150;
          gameOverText.y = startY - 50;
          app.stage.addChild(gameOverText);
          textMap.set("gameOver", gameOverText);
        }

        // Get winning player
        const winningPlayer = gameSession.winningPlayerID ? gameSession.players.get(gameSession.winningPlayerID) : null;

        if (winningPlayer) {
          // Display winner ID
          if (!textMap.has("winner")) {
            const winnerText = new Text({
              text: `Winner: Player ${winningPlayer.id}`,
              style: new TextStyle({
                fontSize: 32,
                fill: winningPlayer.colour,
                fontWeight: 'bold'
              })
            });
            winnerText.x = startX + 150;
            winnerText.y = startY + 20;
            app.stage.addChild(winnerText);
            textMap.set("winner", winnerText);
          }

          // Display score (snake length)
          if (!textMap.has("score")) {
            const score = winningPlayer.body.length;
            const scoreText = new Text({
              text: `Score: ${score}`,
              style: new TextStyle({
                fontSize: 28,
                fill: 'white'
              })
            });
            scoreText.x = startX + 200;
            scoreText.y = startY + 60;
            app.stage.addChild(scoreText);
            textMap.set("score", scoreText);
          }

          // Draw the winning snake at full size
          const snakeStartX = startX + 50;
          const snakeStartY = startY + 120;

          // Draw each body segment
          winningPlayer.body.forEach((bodySegment, index) => {
            const segmentX = snakeStartX + (index * gridSize);
            const segmentY = snakeStartY;

            graphics.rect(segmentX, segmentY, gridSize, gridSize);
            graphics.fill(winningPlayer.colour);
            graphics.stroke({ width: 2, color: "white" });
          });

          // Draw eyes on the head (first segment)
          const headX = snakeStartX;
          const headY = snakeStartY;
          const eyeOffsetSnake = gridSize / 3;

          graphics.circle(headX + eyeOffsetSnake, headY + eyeOffsetSnake, 4);
          graphics.fill("white");
          graphics.circle(headX + eyeOffsetSnake, headY + (eyeOffsetSnake * 2), 4);
          graphics.fill("white");
        }

        // Display refresh instruction
        if (!textMap.has("refresh")) {
          const refreshText = new Text({
            text: 'Refresh to join another session',
            style: new TextStyle({
              fontSize: 24,
              fill: 'yellow',
              fontStyle: 'italic'
            })
          });
          refreshText.x = startX + 100;
          refreshText.y = startY + 500;
          app.stage.addChild(refreshText);
          textMap.set("refresh", refreshText);
        }
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

function removeText(currentState) {
  if (removeTextState === currentState) {
    return
  }

  removeTextState = currentState

  if (textMap.size > 0) {
    for (const [key, textValue] of textMap) {
      if (textValue.parent) {
        textValue.parent.removeChild(textValue);
      }
      textValue.destroy()
      textMap.delete(key);
    }
  }
}