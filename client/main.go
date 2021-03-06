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
	"github.com/sparrc/go-ping"
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
			Name:    "single",
			Aliases: []string{"s"},
			Usage:   "Run a single speed test",
			Action:  single,
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
			},
		},
		{
			Name:    "ping",
			Aliases: []string{"p"},
			Usage:   "Run a series of pings and get statistics",
			Action:  runPing,
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

func single(c *cli.Context) error {
	port := c.Uint("Port")
	length := int(c.Uint("Length"))

	runSpeedTest(c.String("Server"), port, length)

	return nil
}

func runPing(c *cli.Context) error {
	log.Println("Running ping to server plex.hhra.me")

	pinger, err := ping.NewPinger("www.google.com")
	if err != nil {
		panic(err)
	}

	pinger.OnRecv = func(pkt *ping.Packet) {
		fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",
			pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
	}
	pinger.OnFinish = func(stats *ping.Statistics) {
		fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
		fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}

	fmt.Printf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr())
	pinger.Run()

	return nil
}

func runSpeedTest(host string, port uint, length int) float64 {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Printf("Failed to connect to %s port %d", host, port)
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

	speedTest := speedtest.NewSpeedTest(conn, speedtest.DirectionDown)

	go speedTest.ReceiveData()

	for {
		select {
		case <-time.After(5 * time.Second):
			select {
			case speedReport := <-speedTest.ReportChan:
				if speedReport.Time != 0 {
					bytesPerSecond := float64(speedReport.Bytes) / speedReport.Time.Seconds()
					log.Printf("%s/s", humanize.Bytes(uint64(bytesPerSecond)))
				} else {
					log.Printf("No data transferred")
				}
			}
		case status := <-speedTest.StatusChan:
			if status.Status == speedtest.StatusAborted {
				log.Println("Speed test aborted!")
				return 0
			} else if status.Status == speedtest.StatusFinished {
				log.Println("Speed test ended")
				log.Printf("Average bytes per second: %s", humanize.Bytes(uint64(speedTest.Result.AverageSpeed)))
				return float64(speedTest.Result.AverageSpeed)
			} else if status.Status == speedtest.StatusStarted {
				log.Println("Speed test started.")
			}
		}
	}
}
