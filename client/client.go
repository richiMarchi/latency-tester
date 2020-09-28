package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
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

func main() {
	flag.Parse()
	address := flag.Arg(0)
	log.SetFlags(0)
	if *requestBytes < 62 || *responseBytes < 62 {
		log.Fatal("Minimum payload size: 62")
	}

	printLogs(*reps, *logFile, *requestBytes, *responseBytes, *interval, address)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: address, Path: "/echo"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	done := make(chan struct{})

	toolRtt, toolFileErr := os.Create(LogPath + "tool-rtt_" + *logFile)
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	osRtt, osRttFileErr := os.Create(LogPath + "os-rtt_" + *logFile)
	if osRttFileErr != nil {
		log.Fatalf("failed creating file: %s", osRttFileErr)
	}
	tcpStats, tcpStatsFileErr := os.Create(LogPath + "tcp-stats_" + *logFile)
	if tcpStatsFileErr != nil {
		log.Fatalf("failed creating file: %s", tcpStatsFileErr)
	}

	timestampMap := make(map[uint64]time.Time)
	stop := false

	var wgDispatcher sync.WaitGroup
	wgDispatcher.Add(1)

	// Parallel read dispatcher
	go readDispatcher(c, &stop, &wgDispatcher, &done, toolRtt, &timestampMap)

	payload := randomString(*requestBytes - 62 /* offset to set the perfect desired message size */)

	resErr := c.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(int(*responseBytes))))
	if resErr != nil {
		log.Println("write: ", resErr)
		return
	}

	// Parallel os ping and tcp stats handlers
	wgDispatcher.Add(2)
	go customPing(address, &wgDispatcher, &done, osRtt)
	go customSocketStats(address, &wgDispatcher, &done, tcpStats)

	if *reps == 0 {
		infiniteSendLoop(&done, c, &interrupt, &payload, &timestampMap)
	} else {
		sendNTimes(*reps, c, &done, &interrupt, &payload, &timestampMap)
	}

	stop = true
	wgDispatcher.Wait()
}

func customPing(address string,
								wGroup *sync.WaitGroup,
								done *chan struct{},
								outputFile *os.File) {
	defer wGroup.Done()
	defer outputFile.Close()
	for {
		output, _ := exec.Command("ping", strings.Split(address, ":")[0], "-c 1").Output()
		rttMs := string(output)
		if strings.Contains(rttMs, "time=") && strings.Contains(rttMs, " ms") {
			floatMs := rttMs[strings.Index(rttMs, "time=") + 5 : strings.Index(rttMs, " ms")]
			outputFile.WriteString(floatMs + "\n")
		}
		select {
		case <-*done:
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
}

func customSocketStats(address string,
											 wGroup *sync.WaitGroup,
											 done *chan struct{},
											 outputFile *os.File) {
	defer wGroup.Done()
	defer outputFile.Close()
	for {
		output, _ := exec.Command("ss", "-ti", "dst", strings.Split(address, ":")[0]).Output()
		outputFile.WriteString(string(output) + "\n")
		select {
		case <-*done:
			return
		case <-time.After(time.Second):
		}
	}
}
