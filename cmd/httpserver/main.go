package main

import (
	"fmt"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

type Server struct {
	Addr     string
	Port     int
	Listener net.Listener
	State    atomic.Bool
}

func Serve(port int) (*Server, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start listener: %w", err)
	}

	server := &Server{
		Addr:     "localhost",
		Port:     port,
		Listener: listener,
	}
	server.State.Store(true)

	// Start listening in a goroutine
	go server.listen()

	return server, nil
}

func (s *Server) Close() error {
	s.State.Store(false)
	if s.Listener != nil {
		return s.Listener.Close()
	}
	return nil
}

func (s *Server) listen() {
	for s.State.Load() {
		conn, err := s.Listener.Accept()
		if err != nil {
			// Check if server is closed before logging error
			if !s.State.Load() {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	// Set read timeout to prevent hanging connections
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Parse the HTTP request
	req, err := request.RequestFromReader(conn)
	if err != nil {
		// Send a 400 Bad Request response
		writeErrorResponseNew(conn, response.StatusBadRequest, err.Error())
		return
	}

	// Log request
	log.Printf("Received %s request for %s", req.RequestLine.Method, req.RequestLine.RequestTarget)

	// Basic routing
	switch req.RequestLine.RequestTarget {
	case "/":
		writeSuccessResponseNew(conn, "Welcome to the simple HTTP server!")
	case "/time":
		writeSuccessResponseNew(conn, fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC1123)))
	case "/echo":
		if req.Body != nil {
			writeSuccessResponseNew(conn, string(req.Body))
		} else {
			writeSuccessResponseNew(conn, "No body to echo")
		}
	default:
		writeErrorResponseNew(conn, response.StatusBadRequest, "The requested resource was not found on this server")
	}
}

func writeSuccessResponseNew(conn net.Conn, content string) {
	// Write status line
	if err := response.WriteStatusLine(conn, response.StatusOK); err != nil {
		log.Printf("Error writing status line: %v", err)
		return
	}

	// Get and write headers
	headers := response.GetDefaultHeaders(len(content))
	if err := response.WriteHeaders(conn, headers); err != nil {
		log.Printf("Error writing headers: %v", err)
		return
	}

	// Write body
	if _, err := io.WriteString(conn, content); err != nil {
		log.Printf("Error writing body: %v", err)
	}
}

func writeErrorResponseNew(conn net.Conn, statusCode response.StatusCode, message string) {
	// Write status line
	if err := response.WriteStatusLine(conn, statusCode); err != nil {
		log.Printf("Error writing status line: %v", err)
		return
	}

	// Get and write headers
	headers := response.GetDefaultHeaders(len(message))
	if err := response.WriteHeaders(conn, headers); err != nil {
		log.Printf("Error writing headers: %v", err)
		return
	}

	// Write body
	if _, err := io.WriteString(conn, message); err != nil {
		log.Printf("Error writing body: %v", err)
	}
}

const port = 42069

func main() {
	s, err := Serve(port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Printf("Server started on http://localhost:%d", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully shutting down")
}
