package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("messages.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read 8 bytes from the file into a slice of bytes.
	data := make([]byte, 8)
	currentLine := ""

	for {
		count, err := file.Read(data)
		if err == io.EOF {
			// End of file reached, print any remaining content
			if currentLine != "" {
				fmt.Printf("read: %s\n", currentLine)
			}
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			break
		}

		// Convert the bytes to a string and split by newlines
		chunk := string(data[:count])
		parts := strings.Split(chunk, "\n")
		length := len(parts)
		println(length)

		// Process all parts except the last one
		for i := 0; i < len(parts)-1; i++ {
			currentLine += parts[i]
			fmt.Printf("read: %s\n", currentLine)
			currentLine = ""
		}

		// Add the last part to the current line (which might not end with a newline)
		currentLine += parts[len(parts)-1]
	}
}
