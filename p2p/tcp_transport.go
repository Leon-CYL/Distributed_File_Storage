package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCP Peer Inplementation

// TCPPeer represents a remote node over TCP stablished connection.
type TCPPeer struct {
	net.Conn

	// If we dialed and retrieved the conn => outbound = true
	// If we accepted and retrieved the conn => outbound = false
	outbound bool

	wg sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       sync.WaitGroup{},
	}
}

func (tp *TCPPeer) Send(data []byte) error {
	_, err := tp.Conn.Write(data)

	return err
}

func (tp *TCPPeer) CloseStream() {
	tp.wg.Done()
}

// TCP Transport Inplementation
type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	listener net.Listener
	TCPTransportOpts
	rpcCh chan RPC

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcCh:            make(chan RPC, 1024),
	}
}

func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// We can only read from the channel, not write to it.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcCh
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}

func (t *TCPTransport) AcceptAndListen() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	log.Printf("TCP Transport: Listening on %s\n", t.ListenAddr)

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			return
		}

		if err != nil {
			fmt.Printf("TCP Accept Error: %s\n", err)
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	peer := NewTCPPeer(conn, outbound)

	defer func() {
		fmt.Printf("Dropping peer connection: %s\n", err)
		conn.Close()
	}()

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop
	for {
		rpc := RPC{}
		err = t.Decoder.Decode(conn, &rpc)

		if err != nil {
			return
		}

		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream waiting...\n", conn.RemoteAddr())
			peer.wg.Wait()
			fmt.Printf("[%s] Stream closed, continuing read loop\n", conn.RemoteAddr())
			continue
		}

		t.rpcCh <- rpc

	}

}
