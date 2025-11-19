# Before we do the golang backend. build the front end app
FROM node:latest AS frontend

WORKDIR /app/snake-client

COPY snake-client/package.json snake-client/package-lock.json ./
RUN npm install

# Building the app should put it in the /dist folder
COPY snake-client ./
RUN npm run build

# Building the Go server
FROM golang:1.25-alpine AS backend

WORKDIR /app/snake-server

COPY /snake-server ./

RUN go build -o ./bin/server ./cmd/main.go

# /dist folder with frontend app needs to be placed where the server can host it
COPY --from=frontend /app/snake-client/dist /app/snake-server/bin/dist

# Server wants to access images in the ./images folder, set the workdir explicitly
WORKDIR /app/snake-server/bin
ENTRYPOINT ["./server"]

