package main

import (
	"fmt"
	"httpfromtcp/internal/request"
	"net"
	"os"
)

func main() {
	listener, err := net.Listen("tcp", "localhost:42069")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Server listening on localhost:42069")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue // Continue to the next iteration instead of exiting
		}

		fmt.Println("Connection accepted")

		// Get a channel of lines from the connection
		// linesChan := getLinesChannel(conn)

		// When we call request.RequestFromReader(conn), it uses the Read method from net.Conn interface
		req, err := request.RequestFromReader(conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request: %v\n", err)
			conn.Close()
			continue // Continue to the next iteration instead of exiting
		}

		// Range over the channel and print each line
		// for line := range linesChan {
		// 	fmt.Println(line) // Just print the line without "read:" prefix
		// }

		// Print the request details
		fmt.Printf("Method: %s, Target: %s, HTTP Version: %s\n",
			req.RequestLine.Method, req.RequestLine.RequestTarget, req.RequestLine.HttpVersion)

		fmt.Println("Connection closed")
	}
}
