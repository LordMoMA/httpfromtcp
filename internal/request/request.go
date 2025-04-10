package request

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"httpfromtcp/internal/headers"
)

// iota is a special keyword in Go that generates a sequence of integers starting from 0.
// https://go.dev/wiki/Iota
const (
	StateInitialized = iota // Parser state: initialized, is assigned the value 0
	StateParsingRequestLine
	StateParsingHeaders
	StateParsingBody
	StateDone // Parser state: done, is assigned the value 3
)

const bufferSize = 8 // Initial buffer size for reading data

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	state       int // Parser state
}

type RequestLine struct {
	HttpVersion   string // "1.1"
	RequestTarget string // "/coffee"
	Method        string // "GET", "POST", "PATCH", "PUT", or "DELETE"
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := &Request{state: StateInitialized}
	buf := make([]byte, bufferSize)
	readToIndex := 0

	for {
		// If the buffer is full, grow it
		if readToIndex == len(buf) {
			newBuf := make([]byte, len(buf)*2)
			copy(newBuf, buf)
			buf = newBuf
		}

		// Read data into the buffer
		n, err := reader.Read(buf[readToIndex:])
		readToIndex += n // Update the number of bytes read

		// Parse what we have so far, even if we hit EOF
		consumed, parseErr := request.parseAndUpdateState(buf[:readToIndex])
		if parseErr != nil {
			return nil, parseErr
		}

		// Remove parsed data from the buffer
		copy(buf, buf[consumed:])
		readToIndex -= consumed

		// If parsing is done, return the request
		if request.state == StateDone {
			break
		}

		// Now check for errors from the read operation
		if err == io.EOF {
			// We hit EOF but parsing isn't done - this means incomplete request
			// Check if we're in the body parsing phase and have a Content-Length header
			if request.state == StateParsingBody {
				contentLengthStr, ok := request.Headers["content-length"]
				if ok {
					contentLength, convErr := strconv.Atoi(contentLengthStr)
					// Only check for short body if Content-Length > 0
					if convErr == nil && contentLength > 0 && (request.Body == nil || len(request.Body) < contentLength) {
						return nil, errors.New("Body shorter than reported content length")
					}
				}
				// If we have no Content-Length header or the body is complete, we're done
				request.state = StateDone
				break
			}
			return nil, errors.New("incomplete request")
		} else if err != nil {
			// Handle other errors
			return nil, err
		}

		// If we didn't read anything and didn't consume anything, we're stuck
		if n == 0 && consumed == 0 {
			return nil, errors.New("no progress in reading or parsing")
		}
	}

	return request, nil
}

func (r *Request) parseAndUpdateState(data []byte) (int, error) {
	if r.state == StateDone {
		return 0, errors.New("error: trying to read data in a done state")
	}

	totalBytesParsed := 0
	for r.state != StateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return totalBytesParsed, err
		}

		if n == 0 {
			// Need more data
			break
		}

		totalBytesParsed += n
	}

	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case StateInitialized, StateParsingRequestLine:
		return r.parseRequestLine(data)
	case StateParsingHeaders:
		return r.parseHeaders(data)
	case StateParsingBody:
		return r.parseBody(data)
	default:
		return 0, fmt.Errorf("invalid state: %d", r.state)
	}
}

func (r *Request) parseRequestLine(data []byte) (int, error) {
	// At the start of parseRequestLine
	if r.state == StateInitialized {
		r.state = StateParsingRequestLine
	}

	// Find the end of the request line
	lineEnd := strings.Index(string(data), "\r\n")
	if lineEnd == -1 {
		return 0, nil // Need more data
	}

	// Parse request line
	line := string(data[:lineEnd])
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return 0, errors.New("invalid request line: expected 3 parts")
	}

	// Validate the HTTP method
	method := parts[0]
	if !isValidMethod(method) {
		return 0, errors.New("invalid method: expected GET, POST, PATCH, PUT, or DELETE")
	}

	// Validate the request target
	requestTarget := parts[1]
	if requestTarget == "" || !strings.HasPrefix(requestTarget, "/") {
		return 0, errors.New("invalid request target: must start with '/'")
	}

	// Validate the HTTP version
	httpVersion, err := parseHttpVersion(parts[2])
	if err != nil {
		return 0, err
	}

	// Update the request state and request line
	r.RequestLine = RequestLine{
		Method:        method,
		RequestTarget: requestTarget,
		HttpVersion:   httpVersion,
	}

	// After successful parsing, update state
	r.state = StateParsingHeaders
	return lineEnd + 2, nil
}

func (r *Request) parseHeaders(data []byte) (int, error) {
	// Initialize headers if needed
	if r.Headers == nil {
		r.Headers = headers.NewHeaders()
	}

	// Check if we have the end of headers
	emptyLinePos := strings.Index(string(data), "\r\n\r\n")
	if emptyLinePos == -1 {
		// No empty line found yet, try to parse what we have
		var bytesParsed int
		for bytesParsed < len(data) {
			n, done, err := r.Headers.Parse(data[bytesParsed:])
			if err != nil {
				return 0, fmt.Errorf("error parsing headers: %w", err)
			}

			if n == 0 && !done {
				// Need more data
				break
			}

			bytesParsed += n

			if done {
				// Found the end marker within this chunk
				r.state = StateParsingBody
				break
			}
		}

		return bytesParsed, nil
	} else {
		// We have the complete headers section, parse it all
		totalBytesParsed := 0
		headerData := data[:emptyLinePos+4] // Include the ending \r\n\r\n

		for totalBytesParsed < len(headerData) {
			n, done, err := r.Headers.Parse(headerData[totalBytesParsed:])
			if err != nil {
				return 0, fmt.Errorf("error parsing headers: %w", err)
			}

			totalBytesParsed += n

			if done || n == 0 {
				break
			}
		}

		r.state = StateParsingBody
		return emptyLinePos + 4, nil
	}
}

func (r *Request) parseBody(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	contentLengthStr, ok := r.Headers["content-length"]
	if !ok {
		r.state = StateDone
		return 0, nil // No Content-Length is ok, just means no body
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return 0, fmt.Errorf("invalid Content-Length: %w", err)
	}

	// If we haven't initialized the body yet, do so now
	if r.Body == nil {
		r.Body = make([]byte, 0, contentLength)
	}

	// Calculate how many bytes we still need
	bytesNeeded := contentLength - len(r.Body)

	// Calculate how many bytes we can read from the data
	bytesToCopy := len(data)
	if bytesToCopy > bytesNeeded {
		bytesToCopy = bytesNeeded
	}

	// Copy the available bytes
	r.Body = append(r.Body, data[:bytesToCopy]...)

	// If we've read the full body, mark as done
	if len(r.Body) == contentLength {
		r.state = StateDone
		fmt.Printf("Successfully processed the entire length of the data %d\n", bytesToCopy)
		return bytesToCopy, nil
	}

	// Otherwise, we need more data
	return bytesToCopy, nil
}

func isValidMethod(method string) bool {
	switch method {
	case "GET", "POST", "PATCH", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

func parseHttpVersion(version string) (string, error) {
	parts := strings.Split(version, "/")
	if len(parts) != 2 || parts[0] != "HTTP" {
		return "", errors.New("invalid HTTP version: malformed version")
	}
	if parts[1] != "1.1" {
		return "", errors.New("invalid HTTP version: expected 1.1")
	}
	return parts[1], nil
}

// String returns a string representation of the Request in the specified format
func (r *Request) String() string {
	var builder strings.Builder

	// Print request line
	builder.WriteString("Request line:\n")
	builder.WriteString(fmt.Sprintf("- Method: %s\n", r.RequestLine.Method))
	builder.WriteString(fmt.Sprintf("- Target: %s\n", r.RequestLine.RequestTarget))
	builder.WriteString(fmt.Sprintf("- Version: %s\n", r.RequestLine.HttpVersion))

	// Print headers
	builder.WriteString("Headers:\n")
	if len(r.Headers) == 0 {
		builder.WriteString("- No headers\n")
	} else {
		// Sort headers by key for consistent output
		keys := make([]string, 0, len(r.Headers))
		for k := range r.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", k, r.Headers[k]))
		}
	}

	// Print body
	if r.Body != nil && len(r.Body) > 0 {
		builder.WriteString("Body:\n")
		builder.WriteString(fmt.Sprintf("- %s\n", string(r.Body)))
	} else {
		builder.WriteString("Body:\n- No body\n")
	}

	return builder.String()
}
