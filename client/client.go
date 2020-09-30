package main

import (
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

	doneRead := make(chan struct{})
	donePing := make(chan struct{})
	ssHandling := make(chan bool)

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
	stopRead := false
	ssReading := false

	var wg sync.WaitGroup

	// Parallel read dispatcher and ss handler
	go readDispatcher(c, &stopRead, &doneRead, toolRtt, &timestampMap)

	payload := randomString(*requestBytes - 62 /* offset to set the perfect desired message size */)

	resErr := c.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(int(*responseBytes))))
	if resErr != nil {
		log.Println("write: ", resErr)
		return
	}

	// Parallel os ping and tcp stats handlers
	wg.Add(2)
	go getSocketStats(c.UnderlyingConn().(*net.TCPConn), &ssReading, tcpStats, &wg, &ssHandling)
	go customPing(address, &wg, &donePing, osRtt)

	// Start making requests
	if *reps == 0 {
		infiniteSendLoop(c, &interrupt, &payload, &timestampMap, &ssReading, &ssHandling)
	} else {
		sendNTimes(*reps, c, &interrupt, &payload, &timestampMap, &ssReading, &ssHandling)
	}
	// Stop all go routines
	stopRead = true
	close(donePing)
	ssHandling <- false

	// Wait for the go routines to complete their job
	<-doneRead
	wg.Wait()
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
