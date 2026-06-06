package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Leon-CYL/Distributed_File_Storage/server"
)

type HTTPServer struct {
	addr       string
	fileServer *server.FileServer
	server     *http.Server
}

func NewHTTPServer(addr string, fileServer *server.FileServer) *HTTPServer {
	httpServer := &HTTPServer{
		addr:       addr,
		fileServer: fileServer,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", httpServer.handleHealth)
	mux.HandleFunc("/files/", httpServer.handleFiles)

	httpServer.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return httpServer
}

func (s *HTTPServer) Start() error {
	log.Printf("HTTP API listening on %s\n", s.addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Close() error {
	return s.server.Close()
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *HTTPServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/files/")
	if key == "" {
		http.Error(w, "missing file key", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handlePutFile(w, r, key)
	case http.MethodGet:
		s.handleGetFile(w, r, key)
	case http.MethodDelete:
		s.handleDeleteFile(w, r, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handlePutFile(w http.ResponseWriter, r *http.Request, key string) {
	defer r.Body.Close()

	if err := s.fileServer.Store(key, r.Body); err != nil {
		http.Error(w, fmt.Sprintf("failed to store file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("file stored successfully"))
}

func (s *HTTPServer) handleGetFile(w http.ResponseWriter, r *http.Request, key string) {
	reader, err := s.fileServer.Get(key)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get file: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("failed to write file response: %v\n", err)
	}
}

func (s *HTTPServer) handleDeleteFile(w http.ResponseWriter, r *http.Request, key string) {
	if err := s.fileServer.Delete(key); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("file deleted successfully"))
}