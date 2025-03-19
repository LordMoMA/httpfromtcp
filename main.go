package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	file, err := os.Open("messages.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	// Read 8 bytes from the file into a slice of bytes.
	data := make([]byte, 8)

	for {
		count, err := file.Read(data)
		if err == io.EOF {
			break // Exit when we reach the end of the file
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			break
		}
		// Print the bytes read as text to stdout
		fmt.Printf("read: %s\n", data[:count])
	}
	// Close the file.
	file.Close()
}
