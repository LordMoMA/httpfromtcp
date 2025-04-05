package headers

import (
	"errors"
	"strings"
	"unicode"
)

type Headers map[string]string

const (
	headerSeparator = ":"
	crlf            = "\r\n"
)

var (
	ErrInvalidData            = errors.New("invalid headers, expected data")
	ErrInvalidSpacing         = errors.New("invalid spacing header")
	ErrMalformedHeaderLine    = errors.New("malformed header line")
	ErrInvalidHeaderFieldName = errors.New("invalid character in header field name")
)

func isValidHeaderFieldChar(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		strings.ContainsRune("!#$%&'*+-.^_`|~", r)
}

func isValidHeaderFieldName(name string) bool {
	if len(name) == 0 {
		return false
	}

	for _, r := range name {
		if !isValidHeaderFieldChar(r) {
			return false
		}
	}

	return true
}

func NewHeaders() Headers {
	return make(map[string]string)
}

func (h Headers) Get(rawKey string) (string, error) {
	key := strings.ToLower(rawKey)
	val, ok := h[key]
	if !ok {
		return "", errors.New("error finding the value")
	}
	return val, nil
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	if len(data) == 0 {
		return 0, false, ErrInvalidData
	}

	// Check if we've reached the end of headers
	if strings.HasPrefix(string(data), crlf) {
		return 2, true, nil
	}

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

	rawKey := strings.TrimSpace(line[:colonIdx])
	if !isValidHeaderFieldName(rawKey) {
		return 0, false, ErrInvalidHeaderFieldName
	}

	key := strings.ToLower(rawKey)
	value := strings.TrimSpace(line[colonIdx+1:])

	if val, ok := h[key]; ok {
		h[key] = val + ", " + value
	} else {
		h[key] = value
	}

	return lineEnd + 2, false, nil
}
