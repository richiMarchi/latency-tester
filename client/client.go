package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/brucespang/go-tcpinfo"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	_ "golang.org/x/xerrors"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

const LogPath = "/tmp/"

var reps = flag.Uint64("reps", 0, "number of repetitions")
var logFile = flag.String("log", "log.csv", "file to store latency numbers")
var requestBytes = flag.Uint("requestPayload", 64, "bytes of the payload")
var responseBytes = flag.Uint("responsePayload", 64, "bytes of the response payload")
var interval = flag.Uint("interval", 1000, "send interval time (ms)")
var pingIp = flag.String("pingIp", "", "ip to ping")
var https = flag.Bool("tls", false, "true if tls enabled")

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

	printLogs(*reps, *logFile, *requestBytes, *responseBytes, *interval, address)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var u url.URL
	var conn *websocket.Conn
	if *https {
		conf := &tls.Config{ InsecureSkipVerify: true }
		dialer := websocket.Dialer{ TLSClientConfig: conf }
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

	toolRtt, toolFileErr := os.Create(LogPath + "tool-rtt_" + *logFile)
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	toolRtt.WriteString("#e2e-rtt,client-send-ts,server-ts\n")
	osRtt, osRttFileErr := os.Create(LogPath + "os-rtt_" + *logFile)
	if osRttFileErr != nil {
		log.Fatalf("failed creating file: %s", osRttFileErr)
	}
	osRtt.WriteString("#os-rtt,ts\n")
	tcpStats, tcpStatsFileErr := os.Create(LogPath + "tcp-stats_" + *logFile)
	if tcpStatsFileErr != nil {
		log.Fatalf("failed creating file: %s", tcpStatsFileErr)
	}
	tcpStats.WriteString("#message-id,state,ca_state,retransmits,probes,backoff,options,pad_cgo_0,rto,ato,snd_mss," +
		"rcv_mss,unacked,sacked,lost,retrans,fackets,last_data_sent,last_ack_sent,last_data_recv,last_ack_recv,pmtu," +
		"rcv_ssthresh,rtt,rttvar,snd_ssthresh,snd_cwnd,advmss,reordering,rcv_rtt,rcv_space,total_retrans\n")
	tracerouteFile, tracerouteFileErr := os.Create(LogPath + "traceroute_" + *logFile)
	if tracerouteFileErr != nil {
		log.Fatalf("failed creating file: %s", tracerouteFileErr)
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
	wg.Add(3)
	if *https {
		go getSocketStats(getConnFromTLSConn(conn.UnderlyingConn().(*tls.Conn)).(*net.TCPConn),
			&ssReading, tcpStats, &wg, &ssHandling)
	} else {
		go getSocketStats(conn.UnderlyingConn().(*net.TCPConn), &ssReading, tcpStats, &wg, &ssHandling)
	}
	go customTraceroute(*pingIp, &wg, tracerouteFile)
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
	log.Println("Waiting for the routines to complete their tasks...")
	wg.Wait()
}

func customTraceroute(tracerouteIp string,
											wGroup *sync.WaitGroup,
											outputFile *os.File) {
	defer wGroup.Done()
	defer outputFile.Close()
	output, _ := exec.Command("traceroute", tracerouteIp).Output()
	outputFile.WriteString(string(output))
}

func customPing(pingIp string,
								wGroup *sync.WaitGroup,
								done *chan struct{},
								outputFile *os.File) {
	defer wGroup.Done()
	defer outputFile.Close()
	for {
		output, _ := exec.Command("ping", pingIp, "-c 1").Output()
		rttMs := string(output)
		if strings.Contains(rttMs, "time=") && strings.Contains(rttMs, " ms") {
			floatMs := rttMs[strings.Index(rttMs, "time=") + 5 : strings.Index(rttMs, " ms")]
			outputFile.WriteString(floatMs + "," + strconv.FormatInt(getTimestamp().UnixNano(), 10) + "\n")
		}
		select {
		case <-*done:
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
}

func getSocketStats(conn *net.TCPConn,
										ssReading *bool,
										outputFile *os.File,
										wg *sync.WaitGroup,
										ssHandling *chan bool) {
	defer wg.Done()
	defer outputFile.Close()
	var sockOpt []*tcpinfo.TCPInfo
	msgId := 1
	for <- *ssHandling {
		for *ssReading {
			tcpInfo, _ := tcpinfo.GetsockoptTCPInfo(conn)
			sockOpt = append(sockOpt, tcpInfo)
		}
		for i, info := range sockOpt {
			if i == 0 || !cmp.Equal(sockOpt[i], sockOpt[i - 1]) {
				str := fmt.Sprintf("%v", *info)
				str = strings.ReplaceAll(str[1:len(str)-1], " ", ",")
				outputFile.WriteString(strconv.Itoa(msgId) + "," + str + "\n")
			}
		}
		sockOpt = sockOpt[:0]
		msgId += 1
	}
}
