package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mhann/gospeedtest"
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

		speed := make(chan speedtest.BytesPerTime)

		go speedtest.SendData(conn, speed)

		for {
			select {
			case <-time.After(10 * time.Second):
				speedReport, ok := <-speed
				if !ok {
					log.Println("Client disconnected")
					return
				}
				if speedReport.Time != 0 {
					bytesPerSecond := float64(speedReport.Bytes) / speedReport.Time.Seconds()
					log.Printf("%s/s", humanize.Bytes(uint64(bytesPerSecond)))
				} else {
					log.Printf("No data transferred")
				}
			}
		}

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
