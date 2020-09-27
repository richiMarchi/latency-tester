package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DataJSON struct {
	Id      uint64
	ServerTimestamp time.Time
	Payload string
}

const LogPath = ""
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

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

	log.Println("Repetitions:\t", *reps)
	log.Println("Log File:\t", LogPath + *logFile)
	log.Println("Request Bytes:\t", *requestBytes)
	log.Println("Response Bytes:\t", *responseBytes)
	log.Println("Send Interval:\t", *interval)
	log.Println("Address:\t", address)
	log.Println()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: address, Path: "/echo"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	done := make(chan struct{})

	toolRtt, toolFileErr := os.Create(LogPath + *logFile)
	if toolFileErr != nil {
		log.Fatalf("failed creating file: %s", toolFileErr)
	}
	osRtt, osRttFileErr := os.Create(LogPath + "os-rtt_" + *logFile)
	if osRttFileErr != nil {
		log.Fatalf("failed creating file: %s", osRttFileErr)
	}
	/*tcpStats, tcpStatsFileErr := os.Create(LogPath + "tcp-stats_" + *logFile)
	if tcpStatsFileErr != nil {
		log.Fatalf("failed creating file: %s", tcpStatsFileErr)
	}*/

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

	wgDispatcher.Add(1)
	go customPing(address, &wgDispatcher, &done, osRtt)

	if *reps == 0 {
		infiniteSendLoop(&done, c, &interrupt, &payload, &timestampMap)
	} else {
		sendNTimes(*reps, c, &done, &interrupt, &payload, &timestampMap)
	}

	stop = true
	wgDispatcher.Wait()
}

func readDispatcher(c *websocket.Conn,
										stop *bool,
										wgDispatcher *sync.WaitGroup,
										done *chan struct{},
										toolRtt *os.File,
										timestampMap *map[uint64]time.Time,) {
	defer wgDispatcher.Done()
	defer close(*done)
	defer toolRtt.Close()

	var wgReader sync.WaitGroup
	var mux sync.Mutex

	for !*stop || len(*timestampMap) > 1 {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read: ", err)
			return
		}
		wgReader.Add(1)

		// dispatch read
		go singleRead(&wgReader, &message, timestampMap, &mux, toolRtt)
	}
	wgReader.Wait()
}

func singleRead(wgReader *sync.WaitGroup,
								message *[]byte,
								timestampMap *map[uint64]time.Time,
								mux *sync.Mutex,
								toolRtt *os.File) {
	defer wgReader.Done()
	tmpTs := getTimestamp()
	var jsonMap DataJSON
	_ = json.Unmarshal(*message, &jsonMap)
	latency := tmpTs.Sub((*timestampMap)[jsonMap.Id])
	log.Printf("%d.\t%d.%d ms", jsonMap.Id+1, latency.Milliseconds(), latency%time.Millisecond)
	serverTs := jsonMap.ServerTimestamp
	mux.Lock()
	toolRtt.WriteString(strconv.Itoa(int(latency.Milliseconds())) + "." + strconv.Itoa(int(latency%time.Millisecond)))
	if serverTs.UnixNano() != 0 {
		firstLeg := serverTs.Sub((*timestampMap)[jsonMap.Id])
		secondLeg := tmpTs.Sub(serverTs)
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.Itoa(int(firstLeg.Milliseconds())) + "." + strconv.Itoa(int(firstLeg%time.Millisecond)))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.Itoa(int(secondLeg.Milliseconds())) + "." + strconv.Itoa(int(secondLeg%time.Millisecond)))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.Itoa(int(serverTs.UnixNano())))
	}
	toolRtt.WriteString("\n")
	mux.Unlock()
	delete(*timestampMap, jsonMap.Id)
}

func sendNTimes(n uint64,
								c *websocket.Conn,
								done *chan struct{},
								interrupt *chan os.Signal,
								payload *string,
								tsMap *map[uint64]time.Time) {
	var i uint64
	for i = 0; i < n; i++ {
		jsonMap := DataJSON{Id: i, Payload: *payload, ServerTimestamp: time.Time{}}
		marshal, _ := json.Marshal(jsonMap)
		err := c.WriteMessage(websocket.TextMessage, marshal)
		(*tsMap)[i] = getTimestamp()
		if err != nil {
			log.Println("write: ", err)
			return
		}
		select {
		case <-*interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			select {
			case <-*done:
			case <-time.After(time.Second):
			}
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close: ", err)
		return
	}
}

func infiniteSendLoop(done *chan struct{},
											c *websocket.Conn,
											interrupt *chan os.Signal,
											payload *string,
											tsMap *map[uint64]time.Time) {

	ticker := time.NewTicker(time.Duration(*interval) * time.Millisecond)
	defer ticker.Stop()

	var id uint64
	id = 0

	for {
		select {
		case <-*done:
			return
		case <-ticker.C:
			jsonMap := DataJSON{ Id: id, Payload: *payload, ServerTimestamp: time.Time{}}
			marshal, _ := json.Marshal(jsonMap)
			err := c.WriteMessage(websocket.TextMessage, marshal)
			(*tsMap)[id] = getTimestamp()
			id = id + 1
			if err != nil {
				log.Println("write: ", err)
				return
			}
		case <-*interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			select {
			case <-*done:
			case <-time.After(time.Second):
			}
			return
		}
	}
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
