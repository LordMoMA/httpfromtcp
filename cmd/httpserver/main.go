package main

import (
	"bytes"
	"fmt"
	"httpfromtcp/internal/headers" // Import headers package
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
type Handler func(req *request.Request, w *response.Writer)

// Server struct definition remains the same
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
		log.Printf("Raw request data received before error:\n%s", requestData.String())
		log.Printf("Error parsing request: %v", err)

		// Attempt to handle specific error cases based on path if possible
		path := extractPathFromRawRequest(requestData.String())
		if path != "" {
			log.Printf("Extracted path from raw request: %s", path)
			minimalReq := &request.Request{
				RequestLine: request.RequestLine{
					RequestTarget: path,
					Method:        "GET", // Assume GET for error handling
					HttpVersion:   "1.1",
				},
				Headers: make(map[string]string),
				Body:    nil,
			}

			// Use new response Writer
			respWriter := response.NewWriter(conn)
			s.Handler(minimalReq, respWriter)

			// Flush the response
			if err := respWriter.Flush(); err != nil {
				log.Printf("Error flushing response: %v", err)
			}
			return
		}

		// Generic bad request if path extraction failed or wasn't applicable
		respWriter := response.NewWriter(conn)
		respWriter.WriteStatusLine(response.StatusBadRequest)

		// Set headers
		headers := headers.NewHeaders()
		headers.Set("Content-Type", "text/html; charset=utf-8")
		respWriter.WriteHeaders(headers)

		// Write body
		respWriter.WriteBody([]byte("Invalid request format\n"))

		// Flush response
		if err := respWriter.Flush(); err != nil {
			log.Printf("Error flushing response: %v", err)
		}
		return
	}

	// Log successful request parsing
	log.Printf("Received %s request for %s", req.RequestLine.Method, req.RequestLine.RequestTarget)

	// Create response writer and pass to handler
	respWriter := response.NewWriter(conn)

	// Call the handler with the new Writer
	s.Handler(req, respWriter)

	// Flush the response to send it
	if err := respWriter.Flush(); err != nil {
		log.Printf("Error flushing response: %v", err)
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
	// HTML content for responses
	badRequestHTML := `<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`

	serverErrorHTML := `<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`

	successHTML := `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`

	// Define our custom handler with the new signature
	handler := func(req *request.Request, w *response.Writer) {
		log.Printf("Handler called with path: %s", req.RequestLine.RequestTarget)

		// Set up common HTML headers
		htmlHeaders := headers.NewHeaders()
		htmlHeaders.Set("Content-Type", "text/html; charset=utf-8")
		htmlHeaders.Set("Connection", "close")

		switch req.RequestLine.RequestTarget {
		case "/yourproblem":
			log.Printf("Matched /yourproblem route")
			w.WriteStatusLine(response.StatusBadRequest)
			w.WriteHeaders(htmlHeaders)
			w.WriteBody([]byte(badRequestHTML))

		case "/myproblem":
			log.Printf("Matched /myproblem route")
			w.WriteStatusLine(response.StatusServerError)
			w.WriteHeaders(htmlHeaders)
			w.WriteBody([]byte(serverErrorHTML))

		default:
			log.Printf("Using default route")
			w.WriteStatusLine(response.StatusOK)
			w.WriteHeaders(htmlHeaders)
			w.WriteBody([]byte(successHTML))
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
