package speedtest

import (
	"errors"
	"math/rand"
	"net"
	"time"
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
}

type SpeedTest struct {
	Result         SpeedTestResult
	ReportChan     chan BytesPerTime
	Connection     net.Conn
	StatusChan     chan SpeedTestStatus
	DataStreamChan chan BytesPerTime // The raw stream used to report bytes per time on each send of bytes.
	TestComplete   bool
	Direction      Direction
}

type Direction int

const (
	DirectionDown Direction = iota
	DirectionUp
)

type Status int

const (
	StatusStarted Status = iota
	StatusFinished
	StatusAborted
)

func NewSpeedTest(connection net.Conn, direction Direction) *SpeedTest {
	return &SpeedTest{
		ReportChan:     make(chan BytesPerTime),
		StatusChan:     make(chan SpeedTestStatus),
		DataStreamChan: make(chan BytesPerTime),
		Connection:     connection,
		Direction:      direction,
	}
}

func (sp *SpeedTest) SpeedAggregator() {
	go func() {
		aggregatedBytesPerTimeSinceLastReport := BytesPerTime{}
		sp.Result = SpeedTestResult{}

		for {
			select {
			case newReport := <-sp.DataStreamChan:
				aggregatedBytesPerTimeSinceLastReport.Bytes += newReport.Bytes
				aggregatedBytesPerTimeSinceLastReport.Time += newReport.Time

				sp.Result.BytesTransferred += newReport.Bytes
				sp.Result.Duration += newReport.Time
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

func (sp *SpeedTest) ReceiveData(conn net.Conn, reportChannel chan BytesPerTime, length int) error {
	buffer := make([]byte, 1024)

	sp.SpeedAggregator()

	speedTestStart := time.Now()

	sp.StatusChan <- SpeedTestStatus{
		Status: StatusStarted,
	}

	for {
		start := time.Now()

		if time.Since(speedTestStart) > (time.Duration(length) * time.Second) {
			sp.StatusChan <- SpeedTestStatus{
				Status: StatusFinished,
			}
			conn.Close()
			return nil
		}

		w, err := conn.Read(buffer)
		if err != nil {
			return err
		}

		sp.DataStreamChan <- BytesPerTime{
			Bytes: uint(w),
			Time:  time.Since(start),
		}
	}

	return nil
}
