package test

import (
	"testing"
	"time"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
	"github.com/stretchr/testify/assert"
)

func newTCPTransportForTest(listenAddr string) *p2p.TCPTransport {
	opts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}

	return p2p.NewTCPTransport(opts)
}

func TestTCPTransport(t *testing.T) {
	listenAddr := ":3000"
	transport := newTCPTransportForTest(listenAddr)

	assert.Equal(t, ":3000", transport.ListenAddr)

	assert.Nil(t, transport.AcceptAndListen())

	defer transport.Close()
}

func TestTCPTransportAddr(t *testing.T) {
	listenAddr := ":3001"
	transport := newTCPTransportForTest(listenAddr)

	assert.Equal(t, listenAddr, transport.Addr())
}

func TestTCPTransportConsumeReturnsChannel(t *testing.T) {
	transport := newTCPTransportForTest(":3002")

	ch := transport.Consume()

	assert.NotNil(t, ch)
}

func TestTCPTransportAcceptAndListenAndClose(t *testing.T) {
	transport := newTCPTransportForTest(":3003")

	err := transport.AcceptAndListen()
	assert.Nil(t, err)

	err = transport.Close()
	assert.Nil(t, err)
}

func TestTCPTransportDial(t *testing.T) {
	serverTransport := newTCPTransportForTest(":3004")

	err := serverTransport.AcceptAndListen()
	assert.Nil(t, err)

	defer serverTransport.Close()

	clientTransport := newTCPTransportForTest(":0")

	err = clientTransport.Dial(":3004")
	assert.Nil(t, err)

	time.Sleep(time.Millisecond * 100)
}

func TestTCPTransportDialInvalidAddress(t *testing.T) {
	transport := newTCPTransportForTest(":0")

	err := transport.Dial(":9999")

	assert.NotNil(t, err)
}

func TestTCPTransportOnPeerIsCalled(t *testing.T) {
	peerCh := make(chan p2p.Peer, 1)

	serverOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3005",
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer: func(peer p2p.Peer) error {
			peerCh <- peer
			return nil
		},
	}

	serverTransport := p2p.NewTCPTransport(serverOpts)

	err := serverTransport.AcceptAndListen()
	assert.Nil(t, err)

	defer serverTransport.Close()

	clientTransport := newTCPTransportForTest(":0")

	err = clientTransport.Dial(":3005")
	assert.Nil(t, err)

	select {
	case peer := <-peerCh:
		assert.NotNil(t, peer)
	case <-time.After(time.Second):
		t.Fatal("expected OnPeer to be called, but it was not")
	}
}