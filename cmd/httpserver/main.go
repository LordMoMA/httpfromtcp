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

// writeHttpResponse writes the complete HTTP response to the connection.
func writeHttpResponse(conn io.Writer, statusCode response.StatusCode, customHeaders headers.Headers, body []byte) {
	log.Printf("Writing response: status=%d, body_len=%d", statusCode, len(body))

	// Write status line
	if err := response.WriteStatusLine(conn, statusCode); err != nil {
		log.Printf("Error writing status line: %v", err)
		// Attempt to close connection if possible, otherwise just return
		if closer, ok := conn.(io.Closer); ok {
			closer.Close()
		}
		return
	}

	// Prepare headers
	h := response.GetDefaultHeaders(len(body))
	// Override defaults with custom headers if provided
	if customHeaders != nil {
		for k, v := range customHeaders {
			h[k] = v // Assumes customHeaders keys are already lowercase
		}
	}
	// Ensure connection is closed for simplicity in this server
	h["connection"] = "close"

	// Write headers
	if err := response.WriteHeaders(conn, h); err != nil {
		log.Printf("Error writing headers: %v", err)
		if closer, ok := conn.(io.Closer); ok {
			closer.Close()
		}
		return
	}

	// Write body if present
	if len(body) > 0 {
		if _, err := conn.Write(body); err != nil {
			log.Printf("Error writing body: %v", err)
			// Connection likely broken, attempt close
			if closer, ok := conn.(io.Closer); ok {
				closer.Close()
			}
			return
		}
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
			}
			responseBuffer := new(bytes.Buffer) // Use buffer even for error cases
			handlerErr := s.Handler(minimalReq, responseBuffer)
			if handlerErr != nil {
				// Use writeHttpResponse for handler errors triggered by path extraction
				writeHttpResponse(conn, handlerErr.StatusCode, nil, []byte(handlerErr.Message))
				return
			}
			// If handler somehow succeeded despite initial parse error (unlikely but handle)
			writeHttpResponse(conn, response.StatusOK, nil, responseBuffer.Bytes())
			return
		}

		// Generic bad request if path extraction failed or wasn't applicable
		writeHttpResponse(conn, response.StatusBadRequest, nil, []byte("Invalid request format\n"))
		return
	}

	// Log successful request parsing
	log.Printf("Received %s request for %s", req.RequestLine.Method, req.RequestLine.RequestTarget)

	// Create a buffer for the handler to write the response body
	responseBuffer := new(bytes.Buffer)

	// Call the handler
	handlerErr := s.Handler(req, responseBuffer)

	// Handle errors if they occurred
	if handlerErr != nil {
		// Use writeHttpResponse for handler errors
		// Create minimal headers for error response (e.g., text/plain)
		errorHeaders := headers.NewHeaders()
		errorHeaders["content-type"] = "text/plain; charset=utf-8"
		writeHttpResponse(conn, handlerErr.StatusCode, errorHeaders, []byte(handlerErr.Message))
		return
	}

	// No error occurred, write success response using writeHttpResponse
	responseBody := responseBuffer.Bytes()
	writeHttpResponse(conn, response.StatusOK, nil, responseBody) // Pass nil for default headers
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
