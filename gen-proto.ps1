protoc -I ./proto --go_out=./snake-server proto/*.proto 
# Directly copy the proto files to the snake-client directory
Copy-Item -Path "./proto" -Destination "./snake-client/src" -Recurse -Force