package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Leon-CYL/Distributed_File_Storage/api"
	"github.com/Leon-CYL/Distributed_File_Storage/crypto"
	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
	"github.com/Leon-CYL/Distributed_File_Storage/server"
	"github.com/Leon-CYL/Distributed_File_Storage/store"
)

func main() {
	host := flag.String("host", "0.0.0.0", "Host address for the server")
	p2pPort := flag.Int("p2p-addr", 5001, "P2P port for this file server")
	httpPort := flag.Int("http-addr", 8001, "HTTP API port for this file server")
	storageRoot := flag.String("storage", "node_storage", "Storage root folder")
	peersFlag := flag.String("peers", "", "Comma-separated bootstrap peer addresses, for example localhost:5001,localhost:5002")

	flag.Parse()

	if *p2pPort <= 0 || *httpPort <= 0 {
		log.Fatalf("ports must be positive: p2p=%d, http=%d", *p2pPort, *httpPort)
	}

	var peers []string
	if *peersFlag != "" {
		peers = strings.Split(*peersFlag, ",")
	}

	p2pAddr := fmt.Sprintf("%s:%d", *host, *p2pPort)
	httpAddr := fmt.Sprintf("%s:%d", *host, *httpPort)

	fmt.Println("Starting file server node...")
	fmt.Printf("Host: %s\n", *host)
	fmt.Printf("P2P Address: %s\n", p2pAddr)
	fmt.Printf("HTTP Address: %s\n", httpAddr)
	fmt.Printf("Storage Root: %s\n", *storageRoot)
	fmt.Printf("Peers: %v\n", peers)

	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    p2pAddr,
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcp := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := server.FileServerOpts{
		EncryptionKey:     crypto.NewEncryptionKey(),
		StorageRoot:       *storageRoot,
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         tcp,
		BootstrapNodes:    peers,
	}

	fileServer := server.NewFileServer(fileServerOpts)

	tcp.OnPeer = fileServer.OnPeer

	go func() {
		if err := fileServer.Start(); err != nil {
			log.Fatalf("file server error: %v", err)
		}
	}()

	httpServer := api.NewHTTPServer(httpAddr, fileServer)

	if err := httpServer.Start(); err != nil {
		log.Fatalf("http server error: %v", err)
	}
}