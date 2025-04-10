package headers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
	a field-name must contain only:

	Uppercase letters: A-Z
	Lowercase letters: a-z
	Digits: 0-9
	Special characters: !, #, $, %, &, ', *, +, -, ., ^, _, `, |, ~
	and at least a length of 1.

	When parsing a single header like "Host: localhost:42069\r\n", you've successfully parsed one header, but you haven't reached the end of the headers section

	More headers might follow
	Only when you encounter \r\n at the start of input data have you reached the end of all headers
*/

func TestParse(t *testing.T) {
	t.Run("Valid single header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069\r\n")
		fmt.Println("data: ", data)
		fmt.Printf("data: %q, length: %d\n", data, len(data))
		n, done, err := headers.Parse(data)

		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.Equal(t, 23, n)
		assert.False(t, done)
	})

	t.Run("Valid single header with extra whitespace", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Content-Type:   application/json   \r\n")
		fmt.Printf("data %s, length %d\n", data, len(string(data)))
		n, done, err := headers.Parse(data)

		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "application/json", headers["content-type"])
		assert.Equal(t, 37, n)
		assert.False(t, done)
	})

	t.Run("Valid 2 headers with existing headers", func(t *testing.T) {
		// First, add one header
		headers := NewHeaders()
		headers["already-present"] = "value"

		// Parse first header
		data1 := []byte("Content-Type: text/html\r\n")
		fmt.Printf("data %s, length %d\n", data1, len(string(data1)))
		n1, done1, err1 := headers.Parse(data1)
		require.NoError(t, err1)
		assert.Equal(t, 25, n1)
		assert.False(t, done1)

		// Parse second header
		data2 := []byte("Content-Length: 256\r\n")
		fmt.Printf("data2 length %d \n", len(string(data2)))
		n2, done2, err2 := headers.Parse(data2)
		require.NoError(t, err2)
		assert.Equal(t, 21, n2)
		assert.False(t, done2)

		// Verify all headers are present
		assert.Equal(t, "value", headers["already-present"])
		assert.Equal(t, "text/html", headers["content-type"])
		assert.Equal(t, "256", headers["content-length"])
		assert.Equal(t, 3, len(headers))
	})

	t.Run("Valid done", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("\r\nContent after headers")
		n, done, err := headers.Parse(data)

		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.True(t, done)
		assert.Equal(t, 0, len(headers))
	})

	t.Run("Invalid spacing header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host : localhost:42069\r\n")
		n, done, err := headers.Parse(data)

		require.Error(t, err)
		assert.Equal(t, ErrInvalidSpacing, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Empty data", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte{}
		n, done, err := headers.Parse(data)

		require.Error(t, err)
		assert.Equal(t, ErrInvalidData, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Incomplete header line", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069") // Missing CRLF
		n, done, err := headers.Parse(data)

		require.NoError(t, err) // This is not an error, just incomplete
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Malformed header without colon", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("MalformedHeader\r\n")
		n, done, err := headers.Parse(data)

		require.Error(t, err)
		assert.Equal(t, ErrMalformedHeaderLine, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Empty header name", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte(": some-value\r\n")
		n, done, err := headers.Parse(data)

		require.Error(t, err)
		assert.Equal(t, ErrMalformedHeaderLine, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("invalid character in header key", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("H@st: localhost:42069\r\n")
		n, done, err := headers.Parse(data)

		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Handle multiple values for same header", func(t *testing.T) {
		headers := NewHeaders()

		// First header
		data1 := []byte("Set-Person: dave-loves-severance\r\n")
		n1, done1, err1 := headers.Parse(data1)
		require.NoError(t, err1)
		assert.Equal(t, 34, n1)
		assert.False(t, done1)

		// Second header with same key
		data2 := []byte("Set-Person: david-loves-rust\r\n")
		n2, done2, err2 := headers.Parse(data2)
		require.NoError(t, err2)
		assert.Equal(t, 30, n2)
		assert.False(t, done2)

		// Third header with same key
		data3 := []byte("Set-Person: helen-likes-hotels\r\n")
		n3, done3, err3 := headers.Parse(data3)
		require.NoError(t, err3)
		assert.Equal(t, 32, n3)
		assert.False(t, done3)

		// Check concatenated value
		assert.Equal(t, "dave-loves-severance, david-loves-rust, helen-likes-hotels", headers["set-person"])
	})

}
