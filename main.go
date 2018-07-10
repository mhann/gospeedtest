package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	for {
		f, err := os.OpenFile("testresults.csv", os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("Error opening log file.")
		}

		speed := runSpeedTest()

		defer f.Close()
		if _, err = f.WriteString(fmt.Sprintf("%v,%f\n", time.Now().Format(time.RFC3339), speed)); err != nil {
			fmt.Printf("Error writing to log file.")
		}
		time.Sleep(1 * time.Minute)
	}
}

func runSpeedTest() float64 {
	conn, err := net.Dial("tcp", "plex.hhra.me:8888")
	if err != nil {
		fmt.Println("Failed to connect to localhost port 8888")
		return 0
	}

	fmt.Fprintf(conn, "r;500000000;d;\n")
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read response from server")
		return 0
	}

	if response != "a;\n" {
		fmt.Println("Server rejected speed test request")
		return 0
	}

	fmt.Println("Server accepted speed test request")

	lengthInt := 50000000
	bytesReceived := 0
	fmt.Println("Starting speed test")
	start := time.Now()
	for bytesReceived < lengthInt {
		bytesBuffer := make([]byte, 512)
		conn.Read(bytesBuffer)
		bytesReceived += 512
	}
	finish := time.Now()
	fmt.Println("Speed test finished")
	elapsed := finish.Sub(start)
	fmt.Printf("Test took %v\n", elapsed)
	bytesPerSecond := float64(lengthInt) / elapsed.Seconds()
	megabitsPerSecond := (bytesPerSecond * 8) / 1000 / 1000
	fmt.Printf("This gives a speed of %fmbps\n", megabitsPerSecond)
	return megabitsPerSecond
}
