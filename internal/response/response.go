package response

import (
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

// WriteStatusLine writes the HTTP status line to the provided writer
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
func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h["content-length"] = fmt.Sprintf("%d", contentLen)
	h["connection"] = "close"
	h["content-type"] = "text/plain"
	h["date"] = time.Now().Format(time.RFC1123)
	return h
}

// WriteHeaders writes all headers to the provided writer
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
