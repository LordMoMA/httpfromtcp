package headers

import (
	"errors"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	res := map[string]string{}
	return res
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	if len(data) == 0 {
		return 0, false, errors.New("invalid headers, expected data")
	}

	colonIdx := strings.Index(string(data), ":")
	if string(data[colonIdx-1]) == " " {
		return 0, false, errors.New("invalid spacing header")
	}

	lineEnd := strings.Index(string(data), "\r\n\r\n")
	if lineEnd == -1 {
		return 0, false, nil
	}

	count := lineEnd + 1

	return count, true, nil
}
