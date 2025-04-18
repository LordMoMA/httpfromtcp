package response

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"httpfromtcp/internal/headers"
)

type StatusCode int

// fake ENUM in Golang
const (
	StatusOK          StatusCode = 200
	StatusBadRequest  StatusCode = 400
	StatusServerError StatusCode = 500
)

// Writer state enum
const (
	stateInitialized = iota
	stateStatusWritten
	stateHeadersWritten
	stateBodyWritten
	stateChunkedBodyStarted
	stateChunkedBodyDone
	stateTrailersWritten
)

// ErrInvalidWriteState is returned when methods are called in the wrong order
var ErrInvalidWriteState = errors.New("invalid state: operations must be called in order (status, headers, body)")

// Writer encapsulates an HTTP response with methods for sending the
// status line, headers, and body in the correct order
type Writer struct {
	statusCode StatusCode
	headers    headers.Headers
	body       *bytes.Buffer
	writer     io.Writer
	state      int
	chunked    bool
	trailers   headers.Headers
}

// NewWriter creates a new response writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		headers:  headers.NewHeaders(),
		body:     new(bytes.Buffer),
		writer:   w,
		state:    stateInitialized,
		trailers: headers.NewHeaders(),
	}
}

// WriteStatusLine writes the HTTP status line with the provided status code
func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != stateInitialized {
		return ErrInvalidWriteState
	}

	w.statusCode = statusCode
	w.state = stateStatusWritten
	return nil
}

// WriteHeaders writes the provided headers to the response
func (w *Writer) WriteHeaders(h headers.Headers) error {
	if w.state != stateStatusWritten {
		return ErrInvalidWriteState
	}

	// Merge with existing headers or replace if keys match
	for k, v := range h {
		w.headers[k] = v
	}

	// Check if we're using chunked encoding
	if value, exists := w.headers["transfer-encoding"]; exists && value == "chunked" {
		w.chunked = true
	}

	w.state = stateHeadersWritten
	return nil
}

// WriteBody writes the provided bytes to the response body
func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != stateHeadersWritten {
		return 0, ErrInvalidWriteState
	}

	n, err := w.body.Write(p)
	if err != nil {
		return n, err
	}

	w.state = stateBodyWritten
	return n, nil
}

/*
HTTP/1.1 200 OK
Content-Type: text/plain
Transfer-Encoding: chunked

<n>/r/n
<data of length n>/r/n
<n>/r/n
<data of length n>/r/n
<n>/r/n
<data of length n>/r/n
<n>/r/n
<data of length n>/r/n
... repeat ...
0\r\n
\r\n
--------------------------

HTTP/1.1 200 OK
Content-Type: text/plain
Transfer-Encoding: chunked

1E
I could go for a cup of coffee
B
But not Java
12
Never go full Java
0
*/

// WriteChunkedBody writes the provided bytes to the response body using chunked encoding
// Each chunk is prefixed with the chunk size in hexadecimal followed by CRLF
// and is terminated with CRLF
func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	// We can start chunked body mode either right after setting headers
	// or after having written previous chunks
	if w.state != stateHeadersWritten && w.state != stateChunkedBodyStarted {
		return 0, ErrInvalidWriteState
	}

	// Mark that we're using chunked encoding
	w.chunked = true
	w.state = stateChunkedBodyStarted

	// If there's no data to write, don't create a chunk
	if len(p) == 0 {
		return 0, nil
	}

	// Write chunk size in hex followed by CRLF
	chunkSizeHex := fmt.Sprintf("%x", len(p))
	_, err := fmt.Fprintf(w.writer, "%s\r\n", chunkSizeHex)
	if err != nil {
		return 0, err
	}

	// Write the actual chunk data
	n, err := w.writer.Write(p)
	if err != nil {
		return n, err
	}

	// Write the trailing CRLF
	_, err = fmt.Fprint(w.writer, "\r\n")
	if err != nil {
		return n, err
	}

	return n, nil
}

// WriteChunkedBodyDone completes a chunked transfer by writing the final "0\r\n\r\n"
func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != stateChunkedBodyStarted {
		return 0, ErrInvalidWriteState
	}

	// Write the final chunk with zero size
	_, err := fmt.Fprint(w.writer, "0\r\n")
	if err != nil {
		return 0, err
	}

	w.state = stateChunkedBodyDone
	return 0, nil
}

// WriteTrailers writes the provided trailer headers after the chunked body is complete
func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.state != stateChunkedBodyDone {
		return ErrInvalidWriteState
	}

	// Write trailers as headers
	for key, value := range h {
		_, err := fmt.Fprintf(w.writer, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}

	// Write the final empty line
	_, err := fmt.Fprint(w.writer, "\r\n")
	if err != nil {
		return err
	}

	w.state = stateTrailersWritten
	return nil
}

// Flush finalizes and sends the complete HTTP response to the underlying writer
func (w *Writer) Flush() error {
	// Ensure we've at least set a status code and headers
	if w.state < stateStatusWritten {
		return errors.New("cannot flush response: status code not set")
	}

	// If headers haven't been written, do that now with defaults
	if w.state < stateHeadersWritten {
		w.WriteHeaders(headers.NewHeaders())
	}

	// Skip further processing if we're already in chunked mode with trailers
	if w.state == stateTrailersWritten {
		return nil
	}

	// If we're in chunked mode but no trailers were written, write the final CRLF
	if w.state == stateChunkedBodyDone {
		_, err := fmt.Fprint(w.writer, "\r\n")
		return err
	}

	// Get the body as bytes
	bodyBytes := w.body.Bytes()

	// Add or update content-length header based on body size (only if not chunked)
	if !w.chunked {
		w.headers["content-length"] = fmt.Sprintf("%d", len(bodyBytes))
	}

	// Write status line
	var reasonPhrase string
	switch w.statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusServerError:
		reasonPhrase = "Internal Server Error"
	}

	_, err := fmt.Fprintf(w.writer, "HTTP/1.1 %d %s\r\n", w.statusCode, reasonPhrase)
	if err != nil {
		return err
	}

	// Write headers
	for key, value := range w.headers {
		_, err := fmt.Fprintf(w.writer, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}

	// Write empty line to separate headers from body
	_, err = fmt.Fprint(w.writer, "\r\n")
	if err != nil {
		return err
	}

	// Write body if present and not chunked
	if !w.chunked && len(bodyBytes) > 0 {
		_, err = w.writer.Write(bodyBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteStatusLine writes the HTTP status line to the provided writer
// Legacy function maintained for backward compatibility
func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	var reasonPhrase string

	switch statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusServerError:
		reasonPhrase = "Internal Server Error"
	}

	_, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)
	return err
}

// GetDefaultHeaders returns the default headers for all responses
// Legacy function maintained for backward compatibility
func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h["content-length"] = fmt.Sprintf("%d", contentLen)
	h["connection"] = "close"
	h["content-type"] = "text/plain"
	h["date"] = time.Now().Format(time.RFC1123)
	return h
}

// WriteHeaders writes all headers to the provided writer
// Legacy function maintained for backward compatibility
func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for key, value := range headers {
		_, err := fmt.Fprintf(w, "%s: %s\r\n", key, value)
		if err != nil {
			return err
		}
	}

	// Write the empty line that separates headers from body
	_, err := fmt.Fprint(w, "\r\n")
	return err
}
