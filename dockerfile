# Before we do the golang backend. build the front end app
FROM node:latest AS frontend

WORKDIR /app/snake-client

COPY snake-client/package.json snake-client/package-lock.json ./
RUN npm install

# Building the app should put it in the /dist folder
COPY snake-client ./
RUN npm run build

# Building the Go server
FROM golang:1.25-alpine AS builder

WORKDIR /app/snake-server

COPY /snake-server ./

# CGO_ENABLED=0 for a fully static binary, so the final image doesn't need to match glibc/musl
RUN CGO_ENABLED=0 go build -o ./bin/server ./cmd/main.go

# Final image only needs the compiled binary, the images the server reads at runtime, and the built frontend, not the Go toolchain or source tree
FROM alpine:latest

WORKDIR /app/snake-server/bin

COPY --from=builder /app/snake-server/bin/server ./server
COPY --from=builder /app/snake-server/images /app/snake-server/images

# /dist folder with frontend app needs to be placed where the server can host it
COPY --from=frontend /app/snake-client/dist ./dist

EXPOSE 443
ENTRYPOINT ["./server"]
