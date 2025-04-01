package headers

import (
	"errors"
	"strings"
)

type Headers map[string]string

const (
	headerSeparator = ":"
	crlf            = "\r\n"
)

var (
	ErrInvalidData         = errors.New("invalid headers, expected data")
	ErrInvalidSpacing      = errors.New("invalid spacing header")
	ErrMalformedHeaderLine = errors.New("malformed header line")
)

func NewHeaders() Headers {
	return make(map[string]string)
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	if len(data) == 0 {
		return 0, false, ErrInvalidData
	}

	// Check if we've reached the end of headers
	if strings.HasPrefix(string(data), crlf) {
		return 2, true, nil
	}

	// Look for line ending
	lineEnd := strings.Index(string(data), crlf)
	if lineEnd == -1 {
		return 0, false, nil // Need more data
	}

	line := string(data[:lineEnd])

	// Split header line into key and value
	colonIdx := strings.Index(line, headerSeparator)
	if colonIdx == -1 || colonIdx == 0 {
		return 0, false, ErrMalformedHeaderLine
	}

	// Check for invalid spacing before colon
	if strings.HasSuffix(line[:colonIdx], " ") {
		return 0, false, ErrInvalidSpacing
	}

	key := strings.TrimSpace(line[:colonIdx])
	value := strings.TrimSpace(line[colonIdx+1:])

	// Store the header
	h[key] = value

	return lineEnd + 2, false, nil
}
