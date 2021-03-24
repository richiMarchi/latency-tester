package main

import (
	"log"
	"net"
	"time"
)

func getTimestamp() time.Time {
	return time.Now()
}

func printLogs(addr net.Addr,
	responseBytes int) {
	log.Println("Connection established with", addr)
	log.Println("Response payload size =", responseBytes)
}
