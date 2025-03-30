package request

import (
	"errors"
	"io"
	"strings"
)

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	// Read the entire request from the reader
	res, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Split the request into lines
	requestParts := strings.Split(string(res), "\r\n")
	if len(requestParts) < 3 {
		return nil, errors.New("invalid request: insufficient parts")
	}

	// Parse the request line
	requestLine, err := parseRequestLine(requestParts[0])
	if err != nil {
		return nil, err
	}

	return &Request{
		RequestLine: *requestLine,
	}, nil
}

func parseRequestLine(line string) (*RequestLine, error) {
	// Split the request line into parts
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return nil, errors.New("invalid request line: expected 3 parts")
	}

	// Validate the HTTP method
	method := parts[0]
	if !isValidMethod(method) {
		return nil, errors.New("invalid method: expected GET, POST, PATCH, PUT, or DELETE")
	}

	// Validate the request target
	requestTarget := parts[1]
	if requestTarget == "" || !strings.HasPrefix(requestTarget, "/") {
		return nil, errors.New("invalid request target: must start with '/'")
	}

	// Validate the HTTP version
	httpVersion, err := parseHttpVersion(parts[2])
	if err != nil {
		return nil, err
	}

	return &RequestLine{
		Method:        method,
		RequestTarget: requestTarget,
		HttpVersion:   httpVersion,
	}, nil
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
