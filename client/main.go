package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mhann/gospeedtest"
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
			Usage:   "Run continuous speed tests in the background",
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
					Name:  "Length",
					Value: 30,
					Usage: "The length of the speed test in seconds.",
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
		{
			Name:    "ping",
			Aliases: []string{"p"},
			Usage:   "Run a series of pings and get statistics",
			Action:  ping,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
}

func background(c *cli.Context) error {
	log.Println("Starting background logging")

	for {
		f, err := os.OpenFile(c.String("LogFile"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Printf("Error opening log file.")
		}

		port := c.Uint("Port")
		length := int(c.Uint("Length"))
		speed := runSpeedTest(c.String("Server"), port, length)

		defer f.Close()
		if _, err = f.WriteString(fmt.Sprintf("%v,%f\n", time.Now().Format(time.RFC3339), speed)); err != nil {
			log.Printf("Error writing to log file.")
		}

		log.Printf("Sleeping for %d minutes", c.Uint("Delay"))
		time.Sleep(time.Duration(c.Uint("Delay")) * time.Minute)
	}
}

func ping(c *cli.Context) error {
	log.Println("Running ping to server plex.hhra.me")
	return nil
}

func runSpeedTest(host string, port uint, length int) float64 {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Printf("Failed to connect to %s port %d")
		return 0
	}

	fmt.Fprintf(conn, "r;%d;d;\n", length)
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Println("Failed to read response from server")
		return 0
	}

	if response != "a;\n" {
		log.Println("Server rejected speed test request")
		return 0
	}

	log.Println("Server accepted speed test request")

	speed := make(chan speedtest.BytesPerTime)
	go speedtest.ReceiveData(conn, speed, length)

	averageBytesPerSecond := uint64(0)
	reports := 0

	for {
		select {
		case <-time.After(5 * time.Second):
			speedReport, ok := <-speed
			if !ok {
				log.Println("Speed test ended")
				log.Printf("Average bytes per second: %s", humanize.Bytes(averageBytesPerSecond))
				return float64(averageBytesPerSecond)
			}
			if speedReport.Time != 0 {
				bytesPerSecond := float64(speedReport.Bytes) / speedReport.Time.Seconds()
				if reports == 0 {
					averageBytesPerSecond = uint64(bytesPerSecond)
					reports = 1
				} else {
					averageBytesPerSecond = uint64(int(averageBytesPerSecond) + ((int(bytesPerSecond) - int(averageBytesPerSecond)) / (reports + 1)))
					reports++
				}
				log.Printf("%s/s", humanize.Bytes(uint64(bytesPerSecond)))
			} else {
				log.Printf("No data transferred")
			}
		}
	}
}
