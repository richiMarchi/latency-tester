package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

type DataJSON struct {
	Id      uint64
	Payload []byte
}

const LogPath = "/log/"

var reps = flag.Uint64("reps", 0, "number of repetitions")
var logFile = flag.String("log", "log.csv", "file to store latency numbers")
var payloadBytes = flag.Uint("payload", 64, "bytes of the payload")
var interval = flag.Uint("interval", 1000, "send interval time (ms)")

func main() {
	flag.Parse()
	address := flag.Arg(0)
	log.SetFlags(0)

	fmt.Println("Repetitions:\t", *reps)
	fmt.Println("Log File:\t", LogPath + *logFile)
	fmt.Println("Payload Bytes:\t", *payloadBytes)
	fmt.Println("Send Interval:\t", *interval)
	fmt.Println("Address:\t", address)

	fmt.Println()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: address, Path: "/echo"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	done := make(chan struct{})

	results, err := os.Create(LogPath + *logFile)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvWriter := csv.NewWriter(results)
	timestampMap := make(map[uint64]int64)
	stop := false

	var wgDispatcher sync.WaitGroup
	wgDispatcher.Add(1)
	// Parallel read dispatcher
	go func() {
		defer wgDispatcher.Done()
		defer close(done)
		defer results.Close()
		defer csvWriter.Flush()
		defer results.WriteString("\n")

		var wgReader sync.WaitGroup

		firstIteration := true
		for !stop || len(timestampMap) > 0 {

			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read: ", err)
				return
			}
			wgReader.Add(1)
			// dispatch read
			go func() {
				defer wgReader.Done()
				tmpTs := getTimestamp()
				var jsonMap DataJSON
				_ = json.Unmarshal(message, &jsonMap)
				latency := tmpTs - timestampMap[jsonMap.Id]
				delete(timestampMap, jsonMap.Id)
				log.Printf("%d.\t%d.%d ms", jsonMap.Id+1, latency/int64(time.Millisecond), latency%int64(time.Millisecond))
				if !firstIteration {
					results.WriteString(",")
				} else {
					firstIteration = false
				}
				results.WriteString(strconv.Itoa(int(latency/int64(time.Millisecond))) + "." + strconv.Itoa(int(latency%int64(time.Millisecond))))
			}()
		}
		wgReader.Wait()
	}()

	payload := make([]byte, *payloadBytes)


	if *reps == 0 {
		infiniteSendLoop(&done, c, &interrupt, &payload, &timestampMap)
	} else {
		sendNTimes(*reps, c, &done, &interrupt, &payload, &timestampMap)
	}

	stop = true
	wgDispatcher.Wait()
}

func sendNTimes(n uint64,
								c *websocket.Conn,
								done *chan struct{},
								interrupt *chan os.Signal,
								payload *[]byte,
								tsMap *map[uint64]int64) {
	var i uint64
	for i = 0; i < n; i++ {
		jsonMap := DataJSON{Id: i, Payload: *payload}
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
											payload *[]byte,
											tsMap *map[uint64]int64) {

	ticker := time.NewTicker(time.Duration(*interval) * time.Millisecond)
	defer ticker.Stop()

	var id uint64
	id = 0

	for {
		select {
		case <-*done:
			return
		case <-ticker.C:
			(*tsMap)[id] = getTimestamp()
			jsonMap := DataJSON{ Id: id, Payload: *payload}
			id = id + 1
			marshal, _ := json.Marshal(jsonMap)
			err := c.WriteMessage(websocket.TextMessage, marshal)
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

func getTimestamp() int64 {
	return time.Now().UnixNano()
}
