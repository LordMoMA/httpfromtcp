package request

import (
	"errors"
	"io"
	"strings"
)

const (
	StateInitialized = iota // Parser state: initialized
	StateDone               // Parser state: done
)

const bufferSize = 8 // Initial buffer size for reading data

type Request struct {
	RequestLine RequestLine
	state       int // Parser state
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
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
		if err != nil && err != io.EOF {
			return nil, err
		}

		readToIndex += n

		// Parse the data
		consumed, err := request.parseAndUpdateState(buf[:readToIndex])
		if err != nil {
			return nil, err
		}

		// Remove parsed data from the buffer
		copy(buf, buf[consumed:])
		readToIndex -= consumed

		// If parsing is done, return the request
		if request.state == StateDone {
			break
		}

		// If EOF is reached and parsing is not done, return an error
		if err == io.EOF {
			return nil, errors.New("incomplete request")
		}
	}

	return request, nil
}

func (r *Request) parseAndUpdateState(data []byte) (int, error) {
	if r.state == StateDone {
		return 0, errors.New("error: trying to read data in a done state")
	}

	lineEnd := strings.Index(string(data), "\r\n")
	if lineEnd == -1 {
		// Not enough data to parse the request line
		return 0, nil
	}

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
	r.state = StateDone

	return lineEnd + 2, nil // +2 to account for "\r\n"
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
