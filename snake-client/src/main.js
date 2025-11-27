import { Application, Graphics, Text, TextStyle } from "pixi.js";
import { Tuple, EntityTypes } from "./snake";
import { Session, SessionState } from "./session";
import * as protohandler from "./protoHandler.js";

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

  // Calculate canvas size: game board + stats panel + margins
  const canvasWidth = startX + numCols * gridSize + 300 + 100; // board + stats + padding
  const canvasHeight = startY + numRows * gridSize + 100; // board + bottom padding

  // Initialize the application
  await app.init({
    background: "black",
    width: canvasWidth,
    height: canvasHeight,
  });

  // Append the application canvas to the document body
  document.getElementById("pixi-container").appendChild(app.canvas);

  const graphics = new Graphics();

  // Add graphics to stage once at the beginning so it stays at the back
  app.stage.addChild(graphics);

  // Listen for animate update
  app.ticker.add(() => {
    graphics.clear();

    // Handle different session states
    switch (gameSession.state) {
      case SessionState.CONNECTING: {
        // Display "connecting to server" message
        if (!textMap.has("connecting")) {
          const connectingText = new Text({
            text: "Connecting to server...",
            style: new TextStyle({
              fontSize: 24,
              fill: "white",
            }),
          });
          connectingText.x = startX;
          connectingText.y = startY - 50;
          app.stage.addChild(connectingText);
          textMap.set("connecting", connectingText);
        }
        break;
      }
      case SessionState.CONNECTED:
      case SessionState.READY: {
        // remove the previous "connecting" text
        if (textMap.has("connecting")) {
          var text = textMap.get("connecting");
          if (text.parent) {
            text.parent.removeChild(text);
          }
          text.destroy();
          textMap.delete("connecting");
        }

        // remove old text values that may be from old players that disconnected
        textMap.forEach((text, playerID) => {
          if (!gameSession.players.has(playerID)) {
            if (text.parent) {
              text.parent.removeChild(text);
            }
            text.destroy();
            textMap.delete(playerID);
          }
        });

        // Display "Hit Enter to ready up" instruction message
        if (!textMap.has("instructions")) {
          const instructionText = new Text({
            text: "Hit Enter to ready up",
            style: new TextStyle({
              fontSize: 20,
              fill: "yellow",
              fontWeight: "bold",
            }),
          });
          instructionText.x = startX;
          instructionText.y = startY - 80;
          app.stage.addChild(instructionText);
          textMap.set("instructions", instructionText);
        }

        // Display player states using Player.State from players collection
        let yOffset = startY - 50;
        gameSession.players.forEach((player, playerID) => {
          // Map PlayerState enum to readable string
          const stateNames = [
            "UNKNOWN",
            "CONNECTING",
            "CONNECTED",
            "READY",
            "PLAYING",
            "DONE",
          ];
          const stateValue = player.state || 2; // Default to CONNECTED if not set
          const stringValue = stateNames[stateValue] || "UNKNOWN";

          // Check if this is the current player
          const isCurrentPlayer =
            gameSession.playerEntity &&
            player.id === gameSession.playerEntity.id;
          const playerLabel = isCurrentPlayer
            ? `YOU (${playerID})`
            : `Player ${playerID}`;

          if (textMap.has(playerID)) {
            let textValue = textMap.get(playerID);
            textValue.text = `${playerLabel}: ${stringValue}`;
            // Update style for current player
            if (isCurrentPlayer) {
              textValue.style.fill = "yellow";
              textValue.style.fontWeight = "bold";
            }
          } else {
            const stateText = new Text({
              text: `${playerLabel}: ${stringValue}`,
              style: new TextStyle({
                fontSize: 18,
                fill: isCurrentPlayer ? "yellow" : "white",
                fontWeight: isCurrentPlayer ? "bold" : "normal",
              }),
            });
            stateText.x = startX + 40; // Offset to make room for snake icon
            stateText.y = yOffset;
            app.stage.addChild(stateText);
            textMap.set(playerID, stateText);
          }

          // Draw small snake icon beside player status
          const snakeIconX = startX + 10;
          const snakeIconY = yOffset + 5;
          const snakeIconSize = 15;

          // If this is the current player, draw a highlight box around their icon
          if (isCurrentPlayer) {
            graphics.rect(
              snakeIconX - 3,
              snakeIconY - 3,
              snakeIconSize + 6,
              snakeIconSize + 6,
            );
            graphics.stroke({ width: 2, color: "yellow" });
          }

          // Draw snake head (small square)
          graphics.rect(snakeIconX, snakeIconY, snakeIconSize, snakeIconSize);
          graphics.fill(player.colour);
          graphics.stroke({ width: 1, color: player.colour });

          // Draw small eyes on the snake icon
          const eyeSize = 2;
          const eyeOffsetIcon = snakeIconSize / 4;
          graphics.circle(
            snakeIconX + eyeOffsetIcon,
            snakeIconY + eyeOffsetIcon,
            eyeSize,
          );
          graphics.fill("white");
          graphics.circle(
            snakeIconX + snakeIconSize - eyeOffsetIcon,
            snakeIconY + eyeOffsetIcon,
            eyeSize,
          );
          graphics.fill("white");

          yOffset += 25;
        });
        break;
      }
      case SessionState.PLAYING: {
        removeText(gameSession.state);
        // Draw the grid
        for (let row = 0; row < numRows; row++) {
          for (let col = 0; col < numCols; col++) {
            const x = col * gridSize + startX;
            const y = row * gridSize + startY;
            graphics.rect(x, y, gridSize, gridSize);
            graphics.stroke({ width: 2, colour: "white" });
          }
        }

        gameSession.entities.forEach((entity) => {
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
        gameSession.players.forEach((player) => {
          drawFace(graphics, player);
        });

        // Draw player stats panel on the right side
        const statsX = startX + numCols * gridSize + 40;
        const statsY = startY;
        const statsWidth = 250;
        const statsPanelHeight = 400;

        // Draw stats panel background
        graphics.rect(statsX, statsY, statsWidth, statsPanelHeight);
        graphics.fill(0x1a1a1a);
        graphics.stroke({ width: 2, color: 0x333333 });

        // Draw "Players" title
        if (!textMap.has("stats-title")) {
          const titleText = new Text({
            text: "Players",
            style: new TextStyle({
              fontSize: 24,
              fill: "#00ff00",
              fontWeight: "bold",
            }),
          });
          titleText.x = statsX + statsWidth / 2 - 45;
          titleText.y = statsY + 10;
          app.stage.addChild(titleText);
          textMap.set("stats-title", titleText);
        }

        // Draw each player's stats
        let playerYOffset = statsY + 50;
        const playerEntryHeight = 80;

        gameSession.players.forEach((player, playerID) => {
          const isCurrentPlayer =
            gameSession.playerEntity &&
            player.id === gameSession.playerEntity.id;

          // Draw player entry background
          graphics.rect(
            statsX + 10,
            playerYOffset,
            statsWidth - 20,
            playerEntryHeight,
          );
          graphics.fill(isCurrentPlayer ? 0x1a1a00 : 0x0a0a0a);
          graphics.stroke({
            width: isCurrentPlayer ? 2 : 1,
            color: isCurrentPlayer ? 0xffff00 : 0x444444,
          });

          // Draw player name
          const playerNameKey = `stat-name-${playerID}`;
          const playerLabel = isCurrentPlayer
            ? `YOU (${playerID.substring(0, 8)})`
            : `Player ${playerID.substring(0, 8)}`;

          if (!textMap.has(playerNameKey)) {
            const nameText = new Text({
              text: playerLabel,
              style: new TextStyle({
                fontSize: 14,
                fill: isCurrentPlayer ? "#ffff00" : "#ffffff",
                fontWeight: "bold",
              }),
            });
            nameText.x = statsX + 20;
            nameText.y = playerYOffset + 5;
            app.stage.addChild(nameText);
            textMap.set(playerNameKey, nameText);
          } else {
            const nameText = textMap.get(playerNameKey);
            nameText.text = playerLabel;
            nameText.y = playerYOffset + 5;
          }

          // Draw state
          const stateKey = `stat-state-${playerID}`;
          const stateNames = [
            "UNKNOWN",
            "CONNECTING",
            "CONNECTED",
            "READY",
            "PLAYING",
            "DONE",
          ];
          const stateValue = player.state || 4;
          const stateText = stateNames[stateValue] || "UNKNOWN";

          if (!textMap.has(stateKey)) {
            const stateTextObj = new Text({
              text: `State: ${stateText}`,
              style: new TextStyle({
                fontSize: 11,
                fill: "#aaaaaa",
              }),
            });
            stateTextObj.x = statsX + 20;
            stateTextObj.y = playerYOffset + 23;
            app.stage.addChild(stateTextObj);
            textMap.set(stateKey, stateTextObj);
          } else {
            const stateTextObj = textMap.get(stateKey);
            stateTextObj.text = `State: ${stateText}`;
            stateTextObj.y = playerYOffset + 23;
          }

          // Draw score (snake length)
          const scoreKey = `stat-score-${playerID}`;
          const score = player.body ? player.body.length : 0;

          if (!textMap.has(scoreKey)) {
            const scoreTextObj = new Text({
              text: `Score: ${score}`,
              style: new TextStyle({
                fontSize: 11,
                fill: "#aaaaaa",
              }),
            });
            scoreTextObj.x = statsX + 20;
            scoreTextObj.y = playerYOffset + 38;
            app.stage.addChild(scoreTextObj);
            textMap.set(scoreKey, scoreTextObj);
          } else {
            const scoreTextObj = textMap.get(scoreKey);
            scoreTextObj.text = `Score: ${score}`;
            scoreTextObj.y = playerYOffset + 38;
          }

          // Draw mini snake representation
          const miniSnakeX = statsX + 20;
          const miniSnakeY = playerYOffset + 53;
          const miniSegmentSize = 8;
          // Don't want snake icons to sprawl off the box, set the max to be 20
          const numSegmentsToDraw = Math.min(
            player.body ? player.body.length : 20,
          );

          for (let i = 0; i < numSegmentsToDraw; i++) {
            graphics.rect(
              miniSnakeX + i * (miniSegmentSize + 1),
              miniSnakeY,
              miniSegmentSize,
              miniSegmentSize,
            );
            graphics.fill(player.colour);
            graphics.stroke({ width: 1, color: player.colour });
          }

          // Draw eyes on first segment
          if (numSegmentsToDraw > 0) {
            const eyeSize = 1.5;
            const eyeOffsetMini = miniSegmentSize / 4;
            graphics.circle(
              miniSnakeX + eyeOffsetMini,
              miniSnakeY + eyeOffsetMini,
              eyeSize,
            );
            graphics.fill("white");
            graphics.circle(
              miniSnakeX + miniSegmentSize - eyeOffsetMini,
              miniSnakeY + eyeOffsetMini,
              eyeSize,
            );
            graphics.fill("white");
          }

          playerYOffset += playerEntryHeight + 10;
        });

        break;
      }
      case SessionState.DONE: {
        removeText(gameSession.state);

        // Display "Game Over" title
        if (!textMap.has("gameOver")) {
          const gameOverText = new Text({
            text: "GAME OVER",
            style: new TextStyle({
              fontSize: 48,
              fill: "red",
              fontWeight: "bold",
            }),
          });
          gameOverText.x = startX + 150;
          gameOverText.y = startY - 50;
          app.stage.addChild(gameOverText);
          textMap.set("gameOver", gameOverText);
        }

        // Get winning player
        const winningPlayer = gameSession.winningPlayerID
          ? gameSession.players.get(gameSession.winningPlayerID)
          : null;

        if (winningPlayer) {
          // Display winner ID
          if (!textMap.has("winner")) {
            const winnerText = new Text({
              text: `Winner: Player ${winningPlayer.id}`,
              style: new TextStyle({
                fontSize: 32,
                fill: winningPlayer.colour,
                fontWeight: "bold",
              }),
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
                fill: "white",
              }),
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
            const segmentX = snakeStartX + index * gridSize;
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
          graphics.circle(
            headX + eyeOffsetSnake,
            headY + eyeOffsetSnake * 2,
            4,
          );
          graphics.fill("white");
        }

        // Display refresh instruction
        if (!textMap.has("refresh")) {
          const refreshText = new Text({
            text: "Refresh to join another session",
            style: new TextStyle({
              fontSize: 24,
              fill: "yellow",
              fontStyle: "italic",
            }),
          });
          refreshText.x = startX + 100;
          refreshText.y = startY + 500;
          app.stage.addChild(refreshText);
          textMap.set("refresh", refreshText);
        }
        break;
      }
    }
  });
})();

function drawSquare(graphics, col, row, colour) {
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
  var eye1 = new Tuple(x, y);
  var eye2 = new Tuple(x, y);

  switch (player.direction) {
    case 1:
      // Up
      eye1.x = x + eyeOffset;
      eye1.y = y + eyeOffset;
      eye2.x = x + eyeOffset * 2;
      eye2.y = y + eyeOffset;
      break;
    case 2:
      // Down
      eye1.x = x + eyeOffset;
      eye1.y = y + eyeOffset * 2;
      eye2.x = x + eyeOffset * 2;
      eye2.y = y + eyeOffset * 2;
      break;
    case 3:
      // Left
      eye1.x = x + eyeOffset;
      eye1.y = y + eyeOffset;
      eye2.x = x + eyeOffset;
      eye2.y = y + eyeOffset * 2;
      break;
    case 4:
      // Right
      eye1.x = x + eyeOffset * 2;
      eye1.y = y + eyeOffset;
      eye2.x = x + eyeOffset * 2;
      eye2.y = y + eyeOffset * 2;
      break;
  }

  graphics.circle(eye1.x, eye1.y, 3);
  graphics.fill("white");
  graphics.stroke({ width: 2, colour: "white" });

  graphics.circle(eye2.x, eye2.y, 3);
  graphics.fill("white");
  graphics.stroke({ width: 2, colour: "white" });
}

function drawCircle(graphics, col, row, colour) {
  // Circles should be in the center of the square
  const x = (col + 0.5) * gridSize + startX;
  const y = (row + 0.5) * gridSize + startY;

  // Draw the circle
  graphics.circle(x, y, gridSize / 4);
  graphics.fill(colour);
  graphics.stroke({ width: 2, colour: colour });
}

// To handle player input
document.addEventListener("keyup", (event) => {
  switch (gameSession.state) {
    case SessionState.CONNECTED:
    case SessionState.READY:
      handlePlayerReadyUp(event);
      break;
    case SessionState.PLAYING:
      handlePlayerInput(event);
      break;
  }
});

function handlePlayerReadyUp(event) {
  if (event.key == "Enter") {
    protohandler.SendReadyUpMessage(
      gameSession.playerEntity.id,
      gameSession.socket,
    );
    gameSession.state = SessionState.READY;
  }
}

function handlePlayerInput(event) {
  let direction = 0;
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
  protohandler.SendInputMessage(
    gameSession.playerEntity.id,
    direction,
    gameSession.socket,
  );
}

function removeText(currentState) {
  if (removeTextState === currentState) {
    return;
  }

  removeTextState = currentState;

  if (textMap.size > 0) {
    for (const [key, textValue] of textMap) {
      if (textValue.parent) {
        textValue.parent.removeChild(textValue);
      }
      textValue.destroy();
      textMap.delete(key);
    }
  }
}
