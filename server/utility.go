package main

import (
	"log"
	"math/rand"
	"net"
	"time"
)

type DataJSON struct {
	Id              uint64
	ClientTimestamp time.Time
	ServerTimestamp time.Time
	Payload         string
	ResponseSize    uint64
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

func printLogs(addr net.Addr,
	responseBytes int) {
	log.Println("Connection established with", addr)
	log.Println("Response payload size =", responseBytes)
}
