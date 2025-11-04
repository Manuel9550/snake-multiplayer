export const EntityTypes = Object.freeze({
  UNKNOWN: 0,
  WALL: 1,
  POINT: 2,
  BODY: 3
});

export class Tuple {
    x = 0;
    y = 0;
    constructor(x,y) {
        this.x = x;
        this.y = y;
    }
}


export class Snake {
    x = 0;
    y = 0;
    id = ""
    direction = 0
    body = [];
    colour = "white"
    constructor(id,x,y,colour) {
        this.id = id;
        this.x = x;
        this.y = y;
        this.body.push(new Tuple(x,y));
        this.colour = colour;
    }

    getBody() {
        return this.body
    }
}

export class Entity {
    x = 0;
    y = 0;
    id = ""
    playerID = ""
    type = ""
    constructor(id,playerID, x,y,type) {
        this.id = id;
        this.playerID = playerID;
        this.x = x;
        this.y = y;
        this.type = type
    }
}

export class PlayerState {
    playerID = ""
    state = ""
    colour = "white"
    constructor(id, state, colour) {
        this.playerID = id;
        this.state = state;
        this.colour = colour;
    }
}