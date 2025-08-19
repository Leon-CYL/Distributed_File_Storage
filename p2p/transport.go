package p2p

import "net"

// Peer is the interface that represents a remote node.
type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

// Transport is anything that handles the communication between the nodes in the network.
// This can be the form of TCP, UDP, WebSocket, etc.
type Transport interface {
	Addr() string
	AcceptAndListen() error
	Consume() <-chan RPC
	Close() error
	Dial(string) error
}
