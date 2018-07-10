package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func main() {
	ln, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("Failed to start server on port 8888")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting new connection")
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Println("New connection...")

	command, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error recieving command from client")
	}
	commandFields := strings.Split(command, ";")

	switch commandFields[0] {
	case "r":
		fmt.Println("New speed test request received")
		if len(commandFields) != 4 {
			fmt.Println("Incorrect number of arguments!")
		}
		length := commandFields[1]
		direction := commandFields[2]
		fmt.Printf("Length: %s, Direction: %s\n", length, direction)
		fmt.Fprintf(conn, "a;\n, _")

		lengthInt, _ := strconv.Atoi(length)

		bytesSent := 0

		for bytesSent < lengthInt {
			bytesBuffer := make([]byte, 512)
			rand.Read(bytesBuffer)
			conn.Write(bytesBuffer)
			bytesSent += 512
		}

		fmt.Printf("Finished sending data")
	}
}
