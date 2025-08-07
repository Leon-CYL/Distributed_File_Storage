package main

import (
	"fmt"
	"log"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
)

func OnPeer(p p2p.Peer) error {
	fmt.Printf("Doing some logic to test onPeer function\n")
	return nil
}

func main() {

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("Message: %+v\n", msg)
		}
	}()

	if err := tr.AcceptAndListen(); err != nil {
		log.Fatal(err)
	}

	select {}
}
