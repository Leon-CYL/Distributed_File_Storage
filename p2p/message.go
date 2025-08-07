package p2p

import "net"

// Message holds any arbitrary data that being sent over
// each transport between two nodes in the network
type RPC struct {
	From    net.Addr
	Payload []byte
}
