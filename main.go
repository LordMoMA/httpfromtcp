package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

func findPatternInLargeFile(filename, pattern string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err) // The door to our data castle is locked!
	}
	defer file.Close() // Always clean up after yourself—Mom would be proud.

	// Create a scanner for efficient line-by-line reading
	scanner := bufio.NewScanner(file)

	// Increase buffer size for long lines (default is too small)
	const maxCapacity = 512 * 1024 // 512KB—because some people really like long lines
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// Channel for matched lines
	matches := make(chan string, 100) // Buffering helps prevent slow consumers from blocking

	// Error channel (because things go wrong, and we need a place to complain)
	errs := make(chan error, 1)

	// Start a goroutine to process matches
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for match := range matches {
			fmt.Println(match) // This is our grand revelation of the matching lines
		}
	}()

	// Scan the file line by line
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			select {
			case matches <- fmt.Sprintf("Line %d: %s", lineNum, line):
				// Match sent successfully—someone's listening!
			default:
				// Channel full, so let's go old-school and print directly
				fmt.Printf("Line %d: %s\n", lineNum, line)
			}
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		errs <- fmt.Errorf("error scanning file: %w", err) // Our scanner stumbled, report it!
	}

	// Signal completion and wait for the processing goroutine to finish
	close(matches)
	wg.Wait()

	// Check if there was an error and return it if so
	select {
	case err := <-errs:
		return err // We found a problem. Mission failed successfully.
	default:
		return nil // No news is good news!
	}
}
