package speedtest

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/sparrc/go-ping"
)

type BytesPerTime struct {
	Bytes uint
	Time  time.Duration
}

type SpeedTestStatus struct {
	Error  error
	Status Status
}

type SpeedTestResult struct {
	Duration         time.Duration
	BytesTransferred uint
	AverageSpeed     uint
	PeakSpeed        uint
	InitialPing      *ping.Statistics
	DuringPing       *ping.Statistics
	AfterPing        *ping.Statistics
	BufferBloat      uint // Buffer bloat in ms
}

type SpeedTest struct {
	Result         SpeedTestResult
	ReportChan     chan BytesPerTime
	Connection     net.Conn
	StatusChan     chan SpeedTestStatus
	DataStreamChan chan BytesPerTime // The raw stream used to report bytes per time on each send of bytes.
	ControlChan    chan SpeedTestControl
	TestComplete   bool
	Direction      Direction
	Duration       int
}

type Direction int

const (
	DirectionDown Direction = iota
	DirectionUp
)

type Status int

const (
	StatusReady Status = iota
	StatusStarted
	StatusRunningInitialPings
	StatusRunningFinalPings
	StatusFinished
	StatusAborted
)

type SpeedTestControl int

const (
	ControlStart SpeedTestControl = iota
	ControlStop
	ControlAbort
)

func NewSpeedTest(connection net.Conn, direction Direction) *SpeedTest {
	return &SpeedTest{
		ReportChan:     make(chan BytesPerTime),
		StatusChan:     make(chan SpeedTestStatus),
		DataStreamChan: make(chan BytesPerTime),
		Connection:     connection,
		Direction:      direction,
		Duration:       20,
	}
}

func (sp *SpeedTest) SpeedAggregator() {
	go func() {
		aggregatedBytesPerTimeSinceLastReport := BytesPerTime{}
		sp.Result = SpeedTestResult{}

		numberOfReports := 0

		for {
			select {
			case newReport := <-sp.DataStreamChan:
				aggregatedBytesPerTimeSinceLastReport.Bytes += newReport.Bytes
				aggregatedBytesPerTimeSinceLastReport.Time += newReport.Time

				sp.Result.BytesTransferred += newReport.Bytes
				sp.Result.Duration += newReport.Time

				bytesPerSecond := float64(newReport.Bytes) / newReport.Time.Seconds()
				if numberOfReports == 0 {
					sp.Result.AverageSpeed = uint(bytesPerSecond)
					numberOfReports = 1
				} else {
					sp.Result.AverageSpeed = uint(int(sp.Result.AverageSpeed) + ((int(bytesPerSecond) - int(sp.Result.AverageSpeed)) / (numberOfReports + 1)))
					numberOfReports++
				}
			case sp.ReportChan <- aggregatedBytesPerTimeSinceLastReport:
				aggregatedBytesPerTimeSinceLastReport = BytesPerTime{}
			case status := <-sp.StatusChan:
				if status.Status == StatusFinished || status.Status == StatusAborted {
					return
				}
			}
		}
	}()
}

func (sp *SpeedTest) SendData() {
	buffer := make([]byte, 1024)
	bytesRead, err := rand.Read(buffer)
	if err != nil {
		sp.StatusChan <- SpeedTestStatus{
			Error:  err,
			Status: StatusAborted,
		}
		return
	}
	if bytesRead != len(buffer) {
		sp.StatusChan <- SpeedTestStatus{
			Error:  errors.New("Unable to read required number of random bytes."),
			Status: StatusAborted,
		}
		return
	}

	sp.SpeedAggregator()

	for {
		start := time.Now()

		w, err := sp.Connection.Write(buffer)
		if err != nil {
			sp.StatusChan <- SpeedTestStatus{
				Error:  err,
				Status: StatusAborted,
			}
			return
		}

		sp.DataStreamChan <- BytesPerTime{
			Bytes: uint(w),
			Time:  time.Since(start),
		}
	}

	return
}

func (sp *SpeedTest) runPings() *ping.Statistics {
	pinger, err := ping.NewPinger("www.google.com")
	if err != nil {
		panic(err)
	}
	pinger.Count = 10
	pinger.Run()                 // blocks until finished
	stats := pinger.Statistics() // get send/receive/rtt stats

	return stats
}

func RunSpeedTest() {

}

func (sp *SpeedTest) ReceiveData() error {
	buffer := make([]byte, 1024)

	sp.SpeedAggregator()

	speedTestStart := time.Now()

	sp.StatusChan <- SpeedTestStatus{
		Status: StatusReady,
	}

	for control := range sp.ControlChan {
		log.Println("Recieved!")
		if control == ControlStart {
			break
		} else {
			break
		}
	}

	sp.StatusChan <- SpeedTestStatus{
		Status: StatusStarted,
	}

	for {
		start := time.Now()

		if time.Since(speedTestStart) > (time.Duration(sp.Duration) * time.Second) {
			sp.StatusChan <- SpeedTestStatus{
				Status: StatusFinished,
			}
			sp.Connection.Close()
			break
		}

		w, err := sp.Connection.Read(buffer)
		if err != nil {
			return err
		}

		sp.DataStreamChan <- BytesPerTime{
			Bytes: uint(w),
			Time:  time.Since(start),
		}
	}

	sp.Result.AverageSpeed = uint(float64(sp.Result.BytesTransferred) / sp.Result.Duration.Seconds())

	return nil
}
