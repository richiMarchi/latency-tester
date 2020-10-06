package main

import (
	"crypto/tls"
	"flag"
	"github.com/gorilla/websocket"
	_ "golang.org/x/xerrors"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
)

const LogPath = "/tmp/"

var reps = flag.Uint64("reps", 0, "number of repetitions")
var logFile = flag.String("log", "log", "file to store latency numbers")
var requestBytes = flag.Uint("requestPayload", 64, "bytes of the payload")
var responseBytes = flag.Uint("responsePayload", 64, "bytes of the response payload")
var interval = flag.Uint("interval", 1000, "send interval time (ms)")
var pingIp = flag.String("pingIp", "", "ip to ping")
var https = flag.Bool("tls", false, "true if tls enabled")
var traceroute = flag.Bool("traceroute", false, "true if traceroute requested")

func main() {
	flag.Parse()
	address := flag.Arg(0)
	log.SetFlags(0)
	if *requestBytes < 62 || *responseBytes < 62 {
		log.Fatal("Minimum payload size: 62")
	}

	if *pingIp == "" {
		log.Fatalf("Ip to ping required")
	}

	printLogs(*reps, *requestBytes, *responseBytes, *interval, *pingIp, *https, *traceroute, address)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var u url.URL
	var conn *websocket.Conn
	if *https {
		conf := &tls.Config{InsecureSkipVerify: true}
		dialer := websocket.Dialer{TLSClientConfig: conf}
		u = url.URL{Scheme: "wss", Host: address, Path: "/echo"}
		c, _, err := dialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial: ", err)
		}
		conn = c
	} else {
		u = url.URL{Scheme: "ws", Host: address, Path: "/echo"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial: ", err)
		}
		conn = c
	}
	defer conn.Close()

	doneRead := make(chan struct{})
	donePing := make(chan struct{})
	ssHandling := make(chan bool)

	toolRtt, toolFileErr := os.Create(LogPath + "tool-rtt_" + *logFile + ".csv")
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	toolRtt.WriteString("#client-send-timestamp,server-timestamp,e2e-rtt\n")
	osRtt, osRttFileErr := os.Create(LogPath + "os-rtt_" + *logFile + ".csv")
	if osRttFileErr != nil {
		log.Fatalf("failed creating file: %s", osRttFileErr)
	}
	osRtt.WriteString("#timestamp,os-rtt\n")
	tcpStats, tcpStatsFileErr := os.Create(LogPath + "tcp-stats_" + *logFile + ".csv")
	if tcpStatsFileErr != nil {
		log.Fatalf("failed creating file: %s", tcpStatsFileErr)
	}
	tcpStats.WriteString("#timestamp,message-id,state,ca_state,retransmits,probes,backoff,options,pad_cgo_0-0," +
		"pad_cgo_0-1,rto,ato,snd_mss,rcv_mss,unacked,sacked,lost,retrans,fackets,last_data_sent,last_ack_sent," +
		"last_data_recv,last_ack_recv,pmtu,rcv_ssthresh,rtt,rttvar,snd_ssthresh,snd_cwnd,advmss,reordering,rcv_rtt," +
		"rcv_space,total_retrans\n")

	if *traceroute {
		tracerouteFile, tracerouteFileErr := os.Create(LogPath + "traceroute_" + *logFile)
		if tracerouteFileErr != nil {
			log.Fatalf("failed creating file: %s", tracerouteFileErr)
		}

		log.Println("Starting traceroute to", *pingIp+"...")
		customTraceroute(*pingIp, tracerouteFile)
		log.Println("Traceroute completed!")
		log.Println()
	}

	stopRead := false
	ssReading := false

	var wg sync.WaitGroup

	// Variable atomically handled in order to keep track of the packets in the network
	var networkPackets uint64 = 0

	// Parallel read dispatcher and ss handler
	go readDispatcher(conn, &stopRead, &doneRead, toolRtt, &networkPackets)

	payload := randomString(*requestBytes - 62 /* offset to set the perfect desired message size */)

	resErr := conn.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(int(*responseBytes))))
	if resErr != nil {
		log.Println("write: ", resErr)
		return
	}

	// Parallel os ping and tcp stats handlers
	wg.Add(2)
	if *https {
		go getSocketStats(getConnFromTLSConn(conn.UnderlyingConn().(*tls.Conn)).(*net.TCPConn),
			&ssReading, tcpStats, &wg, &ssHandling)
	} else {
		go getSocketStats(conn.UnderlyingConn().(*net.TCPConn), &ssReading, tcpStats, &wg, &ssHandling)
	}
	go customPing(*pingIp, &wg, &donePing, osRtt)

	// Start making requests
	if *reps == 0 {
		infiniteSendLoop(conn, &interrupt, &payload, &ssReading, &ssHandling, &networkPackets)
	} else {
		sendNTimes(*reps, conn, &interrupt, &payload, &ssReading, &ssHandling, &networkPackets)
	}
	// Stop all go routines
	stopRead = true
	close(donePing)
	ssHandling <- false

	// Wait for the go routines to complete their job
	<-doneRead
	wg.Wait()
	log.Println()
	log.Println("Everything is completed!")
}
