package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	lines := make(chan string)

	go func() {
		defer f.Close()

		// Read 8 bytes from the file into a slice of bytes.
		data := make([]byte, 8)
		currentLine := ""

		for {
			count, err := f.Read(data)
			if err == io.EOF {
				// End of file reached, send any remaining content
				if currentLine != "" {
					lines <- currentLine
				}
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading: %v\n", err)
				break
			}

			// Convert the bytes to a string and split by newlines
			chunk := string(data[:count])
			parts := strings.Split(chunk, "\n")

			// Process all parts except the last one
			for i := 0; i < len(parts)-1; i++ {
				currentLine += parts[i]
				lines <- currentLine
				currentLine = ""
			}

			// Add the last part to the current line (which might not end with a newline)
			currentLine += parts[len(parts)-1]
		}

		close(lines)
	}()

	return lines
}
