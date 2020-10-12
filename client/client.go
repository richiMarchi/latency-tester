package main

import (
	"flag"
	"github.com/gorilla/websocket"
	_ "golang.org/x/xerrors"
	"log"
	"os"
	"os/signal"
	"sync"
)

const LogPath = "/tmp/"

var reps = flag.Uint64("reps", 0, "number of repetitions")
var logFile = flag.String("log", "log", "file to store latency numbers")
var requestBytes = flag.Uint64("requestPayload", 64, "bytes of the payload")
var responseBytes = flag.Uint64("responsePayload", 64, "bytes of the response payload")
var interval = flag.Uint64("interval", 1000, "send interval time (ms)")
var https = flag.Bool("tls", false, "true if tls enabled")
var traceroute = flag.Bool("traceroute", false, "true if traceroute requested")
var address string
var pingIp string

func main() {
	flag.Parse()
	address = flag.Arg(0)
	pingIp = flag.Arg(1)
	log.SetFlags(0)
	if *requestBytes < 62 || *responseBytes < 62 {
		log.Fatal("Minimum payload size: 62")
	}

	if address == "" {
		log.Fatal("Server address required")
	}

	if pingIp == "" {
		log.Fatal("Address to ping required")
	}

	printLogs()

	// Handle SIGINT as channel
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Create websocket communication channel
	conn := connect()
	defer conn.Close()

	// File creation
	toolRtt, toolFileErr := os.Create(LogPath + "tool-rtt_" + *logFile + ".csv")
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	toolRtt.WriteString("#client-send-timestamp,server-timestamp,e2e-rtt\n")
	defer toolRtt.Close()
	osRtt, osRttFileErr := os.Create(LogPath + "os-rtt_" + *logFile + ".csv")
	if osRttFileErr != nil {
		log.Fatalf("failed creating file: %s", osRttFileErr)
	}
	osRtt.WriteString("#timestamp,os-rtt\n")
	defer osRtt.Close()
	tcpStats, tcpStatsFileErr := os.Create(LogPath + "tcp-stats_" + *logFile + ".csv")
	if tcpStatsFileErr != nil {
		log.Fatalf("failed creating file: %s", tcpStatsFileErr)
	}
	tcpStats.WriteString("#timestamp,message-id,state,ca_state,retransmits,probes,backoff,options,pad_cgo_0-0," +
		"pad_cgo_0-1,rto,ato,snd_mss,rcv_mss,unacked,sacked,lost,retrans,fackets,last_data_sent,last_ack_sent," +
		"last_data_recv,last_ack_recv,pmtu,rcv_ssthresh,rtt,rttvar,snd_ssthresh,snd_cwnd,advmss,reordering,rcv_rtt," +
		"rcv_space,total_retrans\n")
	defer tcpStats.Close()

	if *traceroute {
		tracerouteFile, tracerouteFileErr := os.Create(LogPath + "traceroute_" + *logFile)
		if tracerouteFileErr != nil {
			log.Fatalf("failed creating file: %s", tracerouteFileErr)
		}

		log.Println("Starting traceroute to", pingIp+"...")
		customTraceroute(pingIp, tracerouteFile)
		tracerouteFile.Close()
		log.Println("Traceroute completed!")
		log.Println()
	}

	// Create synchronization channels
	doneRead := make(chan struct{})
	donePing := make(chan struct{})
	ssHandling := make(chan uint64)
	reset := make(chan *websocket.Conn, 2)

	// Parallel read dispatcher
	go readDispatcher(conn, doneRead, toolRtt, reset)

	var wg sync.WaitGroup
	ssReading := false

	// Parallel os ping and tcp stats handlers
	wg.Add(2)
	go getSocketStats(conn, &ssReading, tcpStats, &wg, ssHandling, reset)
	go customPing(pingIp, &wg, donePing, osRtt)

	// Start making requests
	requestSender(conn, interrupt, &ssReading, ssHandling, reset)

	// Stop all go routines
	close(donePing)
	ssHandling <- 0

	// Wait for the go routines to complete their job
	<-doneRead
	wg.Wait()
	log.Println()
	log.Println("Everything is completed!")
}
