package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
)

func newServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcp := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcp,
		BootstrapNodes:    nodes,
	}

	fs := NewFileServer(fileServerOpts)

	tcp.OnPeer = fs.OnPeer

	return fs
}

func main() {
	s1 := newServer(":3000", "")
	s2 := newServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(time.Second * 1)
	go s2.Start()
	time.Sleep(time.Second * 1)

	// data := bytes.NewReader([]byte("my big data file here!"))
	// s2.Store("text.txt", data)
	// time.Sleep(time.Microsecond * 5)

	r, err := s2.Get("text.txt")
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

}
