package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	operation := os.Args[1]

	switch operation {
	case "get":
		handleGet(os.Args[2:])
	case "put":
		handlePut(os.Args[2:])
	case "delete":
		handleDelete(os.Args[2:])
	case "health":
		handleHealth(os.Args[2:])
	default:
		fmt.Printf("Unknown operation: %s\n", operation)
		printUsage()
	}
}

func handleGet(args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	addr := fs.String("addr", "http://localhost:8001", "HTTP server address")
	key := fs.String("key", "", "Key for the file")
	outPath := fs.String("out", "", "Local file path to save downloaded file")

	fs.Parse(args)

	if *key == "" || *outPath == "" {
		log.Fatal("get requires --key and --out")
	}

	url := fmt.Sprintf("%s/files/%s", *addr, *key)

	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		log.Fatalf("get failed: status=%d body=%s", response.StatusCode, string(body))
	}

	file, err := os.Create(*outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("GET success: wrote file to %s\n", *outPath)
}

func handlePut(args []string) {
	fs := flag.NewFlagSet("put", flag.ExitOnError)

	addr := fs.String("addr", "http://localhost:8001", "HTTP server address")
	key := fs.String("key", "", "Key for the file")
	filePath := fs.String("file", "", "Local file path to upload")

	fs.Parse(args)

	if *key == "" || *filePath == "" {
		log.Fatal("put requires --key and --file")
	}

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	url := fmt.Sprintf("%s/files/%s", *addr, *key)

	request, err := http.NewRequest(http.MethodPut, url, file)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Content-Type", "application/octet-stream")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	if response.StatusCode >= 400 {
		log.Fatalf("put failed: status=%d body=%s", response.StatusCode, string(body))
	}

	fmt.Printf("PUT success: %s\n", string(body))
}

func handleDelete(args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)

	addr := fs.String("addr", "http://localhost:8001", "HTTP server address")
	key := fs.String("key", "", "Key for the file")

	fs.Parse(args)

	if *key == "" {
		log.Fatal("delete requires --key")
	}

	url := fmt.Sprintf("%s/files/%s", *addr, *key)

	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	if response.StatusCode >= 400 {
		log.Fatalf("delete failed: status=%d body=%s", response.StatusCode, string(body))
	}

	fmt.Printf("DELETE success: %s\n", string(body))
}

func handleHealth(args []string) {
	fs := flag.NewFlagSet("health", flag.ExitOnError)

	addr := fs.String("addr", "http://localhost:8001", "HTTP server address")

	fs.Parse(args)

	url := fmt.Sprintf("%s/health", *addr)

	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	if response.StatusCode >= 400 {
		log.Fatalf("health failed: status=%d body=%s", response.StatusCode, string(body))
	}

	fmt.Printf("Health: %s\n", string(body))
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  client put --addr http://localhost:8001 --key <key> --file <path>")
	fmt.Println("  client get --addr http://localhost:8001 --key <key> --out <path>")
	fmt.Println("  client delete --addr http://localhost:8001 --key <key>")
	fmt.Println("  client health --addr http://localhost:8001")
}