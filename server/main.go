package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mhann/gospeedtest"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "gospeedtestserver"
	app.Version = "0.0.1"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Marcus Hann",
			Email: "marcus@hhra.me",
		},
	}
	app.Usage = "Go speedtest server"
	app.EnableBashCompletion = true

	app.Commands = []cli.Command{
		{
			Name:    "start",
			Aliases: []string{"s"},
			Usage:   "Start the speedtest server.",
			Action:  start,
			Flags: []cli.Flag{
				cli.UintFlag{
					Name:  "Port",
					Value: 8888,
					Usage: "The default port for the speedtest server to listen on.",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
}

func start(c *cli.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", c.Uint("Port")))
	if err != nil {
		log.Println("Failed to start server on port 8888")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting new connection")
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	log.Println("New connection...")

	command, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Println("Error recieving command from client")
		return
	}
	commandFields := strings.Split(command, ";")

	switch commandFields[0] {
	case "r":
		log.Println("New speed test request received")
		if len(commandFields) != 4 {
			log.Println("Incorrect number of arguments!")
			return
		}
		length := commandFields[1]
		direction := commandFields[2]
		log.Printf("Length: %s, Direction: %s\n", length, direction)
		fmt.Fprintf(conn, "a;\n, _")

		startSpeedTest(conn)
	}
}

func startSpeedTest(conn net.Conn) {
	speedTest := speedtest.NewSpeedTest(conn, speedtest.DirectionUp)

	go speedTest.SendData()

	for {
		select {
		case <-time.After(10 * time.Second):
			speedReport := <-speedTest.ReportChan
			if speedReport.Time != 0 {
				bytesPerSecond := float64(speedReport.Bytes) / speedReport.Time.Seconds()
				log.Printf("[%s] %s/s", conn.RemoteAddr(), humanize.Bytes(uint64(bytesPerSecond)))
			} else {
				log.Printf("[%s] No data transferred", conn.RemoteAddr())
			}
		case status := <-speedTest.StatusChan:
			if status.Status == speedtest.StatusAborted {
				log.Printf("[%s] Speed test aborted!", conn.RemoteAddr())
				return
			}
		}
	}

	log.Printf("[%s] Finished sending data", conn.RemoteAddr())
}
