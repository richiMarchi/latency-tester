package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var reps = flag.Uint64("reps", 0, "number of repetitions")
var logFile = flag.String("log", "/execdir/log", "file to store latency numbers")
var requestBytes = flag.Uint64("requestPayload", 64, "bytes of the payload")
var responseBytes = flag.Uint64("responsePayload", 64, "bytes of the response payload")
var interval = flag.Uint64("interval", 1000, "send interval time (ms)")
var https = flag.Bool("tls", false, "true if TLS enabled")
var tracerouteIp = flag.String("traceroute", "", "traceroute ip if requested")
var address string

func main() {
	flag.Parse()
	address = flag.Arg(0)
	log.SetFlags(0)
	if *requestBytes < 62 || *responseBytes < 62 {
		log.Fatal("Minimum payload size: 62")
	}

	if address == "" {
		log.Fatal("Server address required")
	}

	if *https {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		address = "https://" + address + "/echo"
	} else {
		address = "http://" + address + "/echo"
	}

	printLogs()

	// Handle SIGINT as channel
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// File creation
	toolRtt, toolFileErr := os.Create(*logFile + ".csv")
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	toolRtt.WriteString("#client-send-timestamp,server-timestamp,e2e-rtt\n")
	defer toolRtt.Close()

	if *tracerouteIp != "" {
		tracerouteFile, tracerouteFileErr := os.Create(*logFile + "_traceroute")
		if tracerouteFileErr != nil {
			log.Fatalf("failed creating file: %s", tracerouteFileErr)
		}

		log.Println("Starting traceroute to", *tracerouteIp+"...")
		customTraceroute(*tracerouteIp, tracerouteFile)
		tracerouteFile.Close()
		log.Println("Traceroute completed!")
		log.Println()
	}

	// Create synchronization channels
	var msgId uint64 = 0

	// Start making requests
	requestSender(interrupt, &msgId, toolRtt)

	log.Println()
	log.Println("Everything is completed!")
}
