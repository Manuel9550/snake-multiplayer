Remove-Item -Path "./snake-server/gen/models" -Recurse -Force
Remove-Item -Path "./snake-client/public/proto" -Recurse -Force

protoc -I ./proto --go_out=./snake-server proto/*.proto 
# Directly copy the proto files to the snake-client directory
Copy-Item -Path "./proto" -Destination "./snake-client/public/proto" -Recurse -Force