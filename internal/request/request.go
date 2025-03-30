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
	res, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	requestParts := strings.Split(string(res), "\r\n")
	if len(requestParts) < 3 {
		return nil, errors.New("Invalid request")
	}

	out := strings.Split(requestParts[0], " ")
	if len(out) < 3 {
		return nil, errors.New("expected 3 parts in request line")
	}

	method := out[0]
	if method != "GET" && method != "POST" && method != "PATCH" && method != "PUT" && method != "DELETE" {
		return nil, errors.New("expected GET or POST method")
	}
	requestTarget := out[1]
	if !strings.Contains(requestTarget, "/") {
		return nil, errors.New("expected request target to contain /")
	}
	httpVersion := strings.Split(out[2], "/")[1]
	if httpVersion == "" || httpVersion != "1.1" {
		return nil, errors.New("expected HTTP version 1.1")
	}

	return &Request{
		RequestLine: RequestLine{
			HttpVersion:   httpVersion,
			RequestTarget: requestTarget,
			Method:        method,
		},
	}, nil
}
