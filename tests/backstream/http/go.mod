module main

go 1.21

require github.com/grepplabs/backstream v0.0.0

require (
	github.com/google/uuid v1.5.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	golang.org/x/net v0.19.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/grepplabs/backstream => ../../..
