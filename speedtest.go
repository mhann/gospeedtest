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

func SpeedAggregator(input chan BytesPerTime, output chan BytesPerTime) {
	go func() {
		aggregatedBytesPerTime := BytesPerTime{}
		for {
			select {
			case newReport, ok := <-input:
				if !ok {
					close(output)
					return
				}
				aggregatedBytesPerTime.Bytes += newReport.Bytes
				aggregatedBytesPerTime.Time += newReport.Time
			case output <- aggregatedBytesPerTime:
				aggregatedBytesPerTime = BytesPerTime{}
			}
		}
	}()
}

func SendData(conn net.Conn, reportChannel chan BytesPerTime) error {
	buffer := make([]byte, 1024)
	bytesRead, err := rand.Read(buffer)
	if err != nil {
		return err
	}
	if bytesRead != len(buffer) {
		return errors.New("Unable to read required number of random bytes.")
	}

	aggregationChannel := make(chan BytesPerTime)

	SpeedAggregator(aggregationChannel, reportChannel)

	for {
		start := time.Now()

		w, err := conn.Write(buffer)
		if err != nil {
			close(aggregationChannel)
			conn.Close()
			return err
		}

		aggregationChannel <- BytesPerTime{
			Bytes: uint(w),
			Time:  time.Since(start),
		}
	}

	return nil
}

func ReceiveData(conn net.Conn, reportChannel chan BytesPerTime, length int) error {
	buffer := make([]byte, 1024)

	aggregationChannel := make(chan BytesPerTime)

	SpeedAggregator(aggregationChannel, reportChannel)

	speedTestStart := time.Now()

	for {
		start := time.Now()

		if time.Since(speedTestStart) > (time.Duration(length) * time.Second) {
			close(aggregationChannel)
			conn.Close()
			return nil
		}

		w, err := conn.Read(buffer)
		if err != nil {
			close(aggregationChannel)
			conn.Close()
			return err
		}

		aggregationChannel <- BytesPerTime{
			Bytes: uint(w),
			Time:  time.Since(start),
		}
	}

	return nil
}
