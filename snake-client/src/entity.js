export class Tuple {
    x = 0;
    y = 0;
    constructor(x,y) {
        this.x = x;
        this.y = y;
    }
}

export class Player {
    x = 0;
    y = 0;
    id = ""
    body = [];
    colour = "white"
    constructor(id,x,y,body) {
        this.id = id;
        this.x = x;
        this.y = y;
        this.body = body;
    }

    getBody() {
        return this.body
    }

    makeBody(bodyData) {
        this.body = bodyData
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