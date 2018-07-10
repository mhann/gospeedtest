package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "gospeedtest"
	app.Version = "0.0.1"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Marcus Hann",
			Email: "marcus@hhra.me",
		},
	}
	app.Usage = "Speed test client and server written in go"

	app.EnableBashCompletion = true

	app.Commands = []cli.Command{
		{
			Name:    "background",
			Aliases: []string{"b", "bg"},
			Usage:   "Run speed tests in the background",
			Action:  background,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "Server",
					Value: "plex.hhra.me",
					Usage: "The server to run the speedtest from.",
				},
				cli.UintFlag{
					Name:  "Port",
					Value: 8888,
					Usage: "The port on which the server software is listening.",
				},
				cli.UintFlag{
					Name:  "Bytes",
					Value: 10000000,
					Usage: "The number of bytes to use for the speed test.",
				},
				cli.UintFlag{
					Name:  "Delay",
					Value: 20,
					Usage: "The delay in minutes between speed test runs.",
				},
				cli.StringFlag{
					Name:  "LogFile",
					Value: "speedtest.csv",
					Usage: "The csv file within which to log speed test results.",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func background(c *cli.Context) error {
	for {
		f, err := os.OpenFile(c.String("LogFile"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("Error opening log file.")
		}

		port := c.Uint("Port")
		data := c.Uint("Bytes")
		speed := runSpeedTest(c.String("Server"), port, data)

		defer f.Close()
		if _, err = f.WriteString(fmt.Sprintf("%v,%f\n", time.Now().Format(time.RFC3339), speed)); err != nil {
			fmt.Printf("Error writing to log file.")
		}
		time.Sleep(time.Duration(c.Uint("Delay")) * time.Minute)
	}
}
func runSpeedTest(host string, port uint, data uint) float64 {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		fmt.Printf("Failed to connect to %s port %d")
		return 0
	}

	fmt.Fprintf(conn, "r;%d;d;\n", data)
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

	lengthInt := data
	bytesReceived := uint(0)
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
