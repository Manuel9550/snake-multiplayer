const keysPressed = {};

document.addEventListener('keydown', (event) => {
    keysPressed[event.code] = true;
});

document.addEventListener('keyup', (event) => {
    keysPressed[event.code] = false;
});

export function GetPlayerInput() {
    // Only return one player input at a time, for simplicity.
    if (keysPressed["KeyW"] === true) {
        return 1
    }
    if (keysPressed["KeyS"] === true) {
        return 2
    }
    if (keysPressed["KeyA"] === true) {
        return 3
    }
    if (keysPressed["KeyD"] === true) {
        return 3
    }
    return 0
}
