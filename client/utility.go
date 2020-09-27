package main

import (
	"log"
	"math/rand"
	"time"
)

type DataJSON struct {
	Id      uint64
	ServerTimestamp time.Time
	Payload string
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func getTimestamp() time.Time {
	return time.Now()
}

func stringWithCharset(length uint, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length uint) string {
	return stringWithCharset(length, charset)
}

func printLogs(reps uint64,
							 logFile string,
							 requestBytes uint,
							 responseBytes uint,
							 interval uint,
							 address string) {
	log.Println("Repetitions:\t", reps)
	log.Println("Log File:\t", LogPath + logFile)
	log.Println("Request Bytes:\t", requestBytes)
	log.Println("Response Bytes:\t", responseBytes)
	log.Println("Send Interval:\t", interval)
	log.Println("Address:\t", address)
	log.Println()
}
