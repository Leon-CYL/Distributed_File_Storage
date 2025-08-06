package p2p

// Peer is the interface that represents a remote node.
type Peer interface {}

// Transport is anything that handles the communication between the nodes in the network.
// This can be the form of TCP, UDP, WebSocket, etc.
type Transport interface {
	AcceptAndListen() error
}