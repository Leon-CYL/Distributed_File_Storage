package main

import (
	"fmt"
	"log"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
)

func main() {

	tr := p2p.NewTCPTransport(":3000")

	if err := tr.AcceptAndListen(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Hello, World!")

	select {}
}
