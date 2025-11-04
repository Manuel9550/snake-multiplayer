protoc -I ./proto --go_out=./snake-server proto/*.proto 
# Directly copy the proto files to the snake-client directory
Copy-Item -Path "./proto" -Destination "./snake-client/src" -Recurse -Force
# Generate the javascript protobuf code
protoc -I ./proto --js_out=import_style=commonjs,binary:./snake-client/src/gen proto/*.proto 