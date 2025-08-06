package p2p

import (
	"bytes"
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents a remote node over TCP stablished connection.
type TCPPeer struct {
	conn net.Conn

	// If we dialed and retrieved the conn => outbound = true
	// If we accepted and retrieved the conn => outbound = false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddr    string
	listener      net.Listener
	shakeHands HandshakeFunc
	decoder       Decoder

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		shakeHands: NOPHandshake,
		listenAddr:    listenAddr,
	}
}

func (t *TCPTransport) AcceptAndListen() error {
	var err error

	t.listener, err = net.Listen("tcp", t.listenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	return nil
}

func (t *TCPTransport) startAcceptLoop() error {
	for {
		conn, err := t.listener.Accept()

		if err != nil {
			fmt.Printf("TCP Accept Error: %s\n", err)
		}

		go t.handleConn(conn)
	}
}

type temp struct {}

func (t *TCPTransport) handleConn(conn net.Conn) {

	peer := NewTCPPeer(conn, false)

	if err := t.shakeHands(conn); err != nil {
		//
	}

	msg := temp{}
	// Read loop
	for {
		if err := t.decoder.Decode(conn, &msg); err != nil {
			fmt.Printf("TCP Transport Error: %s\n", err)
			continue
		}

	}

}
