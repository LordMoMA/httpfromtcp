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
		fmt.Println(req)

		fmt.Println("Connection closed")
	}
}

/*
When you use fmt.Println(req), it can print the format defined in func (r *Request) String() string because of how Go's printing functions work with custom types. This is a fundamental feature of Go's interface system.

How It Works
The Stringer Interface: Go has a built-in interface in the fmt package called Stringer:

112
Automatic Interface Implementation: When you define a String() string method for your Request type, it automatically implements the Stringer interface (without explicitly declaring it).

The fmt Package Behavior: When you pass any value to functions like fmt.Println(), fmt.Printf(), etc., the fmt package checks if that value implements the Stringer interface.

Custom String Representation: If the value implements Stringer (by having a String() method), fmt will call that method to get a string representation of your object instead of using the default formatting.
*/
