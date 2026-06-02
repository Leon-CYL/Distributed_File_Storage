package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
	"github.com/Leon-CYL/Distributed_File_Storage/server"
	"github.com/Leon-CYL/Distributed_File_Storage/crypto"
	"github.com/Leon-CYL/Distributed_File_Storage/store"
	
)

func newServer(listenAddr string, nodes ...string) *server.FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcp := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := server.FileServerOpts{
		EncryptionKey:     crypto.NewEncryptionKey(),
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         tcp,
		BootstrapNodes:    nodes,
	}

	fs := server.NewFileServer(fileServerOpts)

	tcp.OnPeer = fs.OnPeer

	return fs
}

func main() {
	s1 := newServer(":3001", "")
	s2 := newServer(":3002", "")
	s3 := newServer(":3000", ":3001", ":3002")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(time.Millisecond * 500)

	go func() {
		log.Fatal(s2.Start())
	}()

	time.Sleep(time.Millisecond * 500)

	go func() {
		log.Fatal(s3.Start())
	}()

	time.Sleep(time.Millisecond * 500)

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("text_%d.txt", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		s3.Store(key, data)

		time.Sleep(time.Millisecond * 5)

		if err := s3.Delete(key); err != nil {
			log.Fatal(err)
		}

		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}
	}

}
