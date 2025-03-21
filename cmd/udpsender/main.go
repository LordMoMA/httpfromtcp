package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		// Print a prompt character
		fmt.Print("> ")

		// Read a line from stdin
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		// Write the line to the UDP connection
		count, err := conn.Write([]byte(line))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Sent %d bytes\n", count)
	}
}
