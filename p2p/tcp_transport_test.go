package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listenAddr := ":3000"
	opts := TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: NOPHandshake,
		Decoder:       DefaultDecoder{},
	}
	transport := NewTCPTransport(opts)

	assert.Equal(t, transport.ListenAddr, ":3000")

	assert.Nil(t, transport.AcceptAndListen())
}

//
