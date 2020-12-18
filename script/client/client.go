package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"os/signal"
	"sync"
)

const LogPath = "/execdir/"

var reps = flag.Uint64("reps", 0, "number of repetitions")
var logFile = flag.String("log", "log", "file to store latency numbers")
var requestBytes = flag.Uint64("requestPayload", 64, "bytes of the payload")
var responseBytes = flag.Uint64("responsePayload", 64, "bytes of the response payload")
var interval = flag.Uint64("interval", 1000, "send interval time (ms)")
var https = flag.Bool("tls", false, "true if TLS enabled")
var tracerouteIp = flag.String("traceroute", "", "traceroute ip if requested")
var sockOpt = flag.Bool("tcpStats", false, "true if TCP Stats requested")
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

	printLogs()

	// Handle SIGINT as channel
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Create websocket communication channel
	conn := connect()
	defer conn.Close()

	// File creation
	toolRtt, toolFileErr := os.Create(LogPath + "" + *logFile + ".csv")
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	toolRtt.WriteString("#client-send-timestamp,server-timestamp,e2e-rtt\n")
	defer toolRtt.Close()

	if *tracerouteIp != "" {
		tracerouteFile, tracerouteFileErr := os.Create(LogPath + "traceroute_" + *logFile)
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
	doneRead := make(chan struct{})
	reset := make(chan *websocket.Conn, 2)

	// Parallel read dispatcher
	go readDispatcher(conn, doneRead, toolRtt, reset)

	var wg sync.WaitGroup
	ssReading := true
	var msgId uint64 = 0

	// If explicitly requested tcp stats handlers
	if *sockOpt {
		tcpStats, tcpStatsFileErr := os.Create(LogPath + "tcp-stats_" + *logFile + ".csv")
		if tcpStatsFileErr != nil {
			log.Fatalf("failed creating file: %s", tcpStatsFileErr)
		}
		tcpStats.WriteString("#timestamp,message-id,state,ca_state,retransmits,probes,backoff,options,pad_cgo_0-0," +
			"pad_cgo_0-1,rto,ato,snd_mss,rcv_mss,unacked,sacked,lost,retrans,fackets,last_data_sent,last_ack_sent," +
			"last_data_recv,last_ack_recv,pmtu,rcv_ssthresh,rtt,rttvar,snd_ssthresh,snd_cwnd,advmss,reordering,rcv_rtt," +
			"rcv_space,total_retrans\n")
		wg.Add(1)
		go getSocketStats(conn, &ssReading, tcpStats, &wg, &msgId, reset)
	}

	// Start making requests
	requestSender(conn, interrupt, &ssReading, reset, &msgId)

	// Stop all go routines
	ssReading = false

	// Wait for the go routines to complete their job
	<-doneRead
	wg.Wait()
	log.Println()
	log.Println("Everything is completed!")
}
