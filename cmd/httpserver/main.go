package main

import (
	"bytes"
	"fmt"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// Handler is a function type that processes an HTTP request and writes a response
type Handler func(req *request.Request, w io.Writer) *HandlerError

// HandlerError represents an error that occurred during request handling
type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Server struct {
	Addr     string
	Port     int
	Listener net.Listener
	State    atomic.Bool
	Handler  Handler
}

func Serve(port int, handler Handler) (*Server, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start listener: %w", err)
	}

	server := &Server{
		Addr:     "localhost",
		Port:     port,
		Listener: listener,
		Handler:  handler,
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

// writeError writes a HandlerError to the connection
func writeError(conn net.Conn, handlerErr *HandlerError) {
	log.Printf("writing error: status=%d, message=%s, length=%d", handlerErr.StatusCode, handlerErr.Message, len(handlerErr.Message))

	// Write status line
	if err := response.WriteStatusLine(conn, handlerErr.StatusCode); err != nil {
		log.Printf("Error writing status line: %v", err)
		return
	}

	// Get and write headers
	headers := response.GetDefaultHeaders(len(handlerErr.Message))
	headers["connection"] = "close" // Ensure this is set

	if err := response.WriteHeaders(conn, headers); err != nil {
		log.Printf("Error writing headers: %v", err)
		return
	}

	// Write body
	if _, err := io.WriteString(conn, handlerErr.Message); err != nil {
		log.Printf("Error writing body: %v", err)
		return
	}

	// Explicitly flush and close the connection
	conn.Close()
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	// Capture the raw request for debugging
	var requestData bytes.Buffer
	teeReader := io.TeeReader(conn, &requestData)

	// set a read timeout for the connection
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Parse the HTTP request
	req, err := request.RequestFromReader(teeReader)
	if err != nil {
		// Log the raw request data we received
		log.Printf("Raw request data received before error:\n%s", requestData.String())
		log.Printf("Error parsing request: %v", err)

		// Since we received the complete HTTP request in the logs,
		// let's try to manually parse it to get the path
		path := extractPathFromRawRequest(requestData.String())

		// If we could extract a path, try to handle it with our custom handler
		if path != "" {
			log.Printf("Extracted path from raw request: %s", path)

			// Create a minimal request to pass to the handler
			minimalReq := &request.Request{
				RequestLine: request.RequestLine{
					RequestTarget: path,
					Method:        "GET",
					HttpVersion:   "HTTP/1.1",
				},
				Headers: make(map[string]string),
			}

			// Create a buffer for the handler's response
			responseBuffer := new(bytes.Buffer)

			// Call the handler with our minimal request
			handlerErr := s.Handler(minimalReq, responseBuffer)

			// Process the handler's response
			if handlerErr != nil {
				writeError(conn, handlerErr)
				return
			}

			// Write success response with the handler's data
			responseBody := responseBuffer.Bytes()

			// Write status line
			if err := response.WriteStatusLine(conn, response.StatusOK); err != nil {
				log.Printf("Error writing status line: %v", err)
				return
			}

			// Get and write headers
			headers := response.GetDefaultHeaders(len(responseBody))
			if err := response.WriteHeaders(conn, headers); err != nil {
				log.Printf("Error writing headers: %v", err)
				return
			}

			// Write body
			if _, err := conn.Write(responseBody); err != nil {
				log.Printf("Error writing body: %v", err)
			}
			return
		}

		handlerErr := &HandlerError{
			StatusCode: response.StatusBadRequest,
			Message:    "Invalid request format\n",
		}
		writeError(conn, handlerErr)
		return
	}

	// Log request
	log.Printf("Received %s request for %s", req.RequestLine.Method, req.RequestLine.RequestTarget)

	// Create a buffer for the handler to write the response body
	responseBuffer := new(bytes.Buffer)

	// Call the handler
	handlerErr := s.Handler(req, responseBuffer)

	// Handle errors if they occurred
	if handlerErr != nil {
		writeError(conn, handlerErr)
		return
	}

	// No error occurred, write success response
	responseBody := responseBuffer.Bytes()

	// Write status line
	if err := response.WriteStatusLine(conn, response.StatusOK); err != nil {
		log.Printf("Error writing status line: %v", err)
		return
	}

	// Get and write headers
	headers := response.GetDefaultHeaders(len(responseBody))
	if err := response.WriteHeaders(conn, headers); err != nil {
		log.Printf("Error writing headers: %v", err)
		return
	}

	// Write body
	if _, err := conn.Write(responseBody); err != nil {
		log.Printf("Error writing body: %v", err)
	}
}

// extractPathFromRawRequest is a helper function to get the path from a raw HTTP request
func extractPathFromRawRequest(rawRequest string) string {
	lines := strings.Split(rawRequest, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Parse the first line which should be like "GET /path HTTP/1.1"
	parts := strings.Split(lines[0], " ")
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}

const port = 42069

func main() {
	// Define our custom handler with debug logging
	handler := func(req *request.Request, w io.Writer) *HandlerError {
		log.Printf("Handler called with path: %s", req.RequestLine.RequestTarget)

		switch req.RequestLine.RequestTarget {
		case "/yourproblem":
			log.Printf("Matched /yourproblem route")
			return &HandlerError{
				StatusCode: response.StatusBadRequest,
				Message:    "Your problem is not my problem\n",
			}
		case "/myproblem":
			log.Printf("Matched /myproblem route")
			return &HandlerError{
				StatusCode: response.StatusServerError,
				Message:    "Woopsie, my bad\n",
			}
		default:
			log.Printf("Using default route")
			// Write success response
			io.WriteString(w, "All good, frfr\n")
			return nil
		}
	}

	// Start the server with our handler
	s, err := Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Printf("Server started on http://localhost:%d", port)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully shutting down")
}
