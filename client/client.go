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
	"time"
)

type DataJSON struct {
	Timestamp int64
	Payload []byte
}

var reps = flag.Int("reps", 0, "number of repetitions")
var logFile = flag.String("log", "log.csv", "file to store latency numbers")
var payloadBytes = flag.Int("payload", 64, "bytes of the payload")
var interval = flag.Int("interval", 1000, "send interval time (ms)")

func main() {
	flag.Parse()
	address := flag.Arg(0)
	if *reps < 0 {
		fmt.Fprintln(os.Stderr, "<repetitions> must be a positive number")
		os.Exit(1)
	}
	if *payloadBytes < 0 {
		fmt.Fprintln(os.Stderr, "<payload-Bytes> must be a positive number")
		os.Exit(1)
	}
	log.SetFlags(0)

	fmt.Println("Repetitions:\t", *reps)
	fmt.Println("Log File:\t", *logFile)
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

	results, err := os.Create(*logFile)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvWriter := csv.NewWriter(results)
	firstIteration := true

	// Parallel read loop
	go func() {
		defer close(done)
		defer results.Close()
		defer csvWriter.Flush()
		defer results.WriteString("\n")
		for  {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read: ", err)
				return
			}
			var jsonMap DataJSON
			_ = json.Unmarshal(message, &jsonMap)
			latency := getTimestamp() - jsonMap.Timestamp
			log.Printf("latency:\t%d.%d ms", latency / int64(time.Millisecond), latency % int64(time.Millisecond))
			if !firstIteration {
				results.WriteString(",")
			} else {
				firstIteration = false
			}
			results.WriteString(strconv.Itoa(int(latency / int64(time.Millisecond))) + "." + strconv.Itoa(int(latency % int64(time.Millisecond))))
		}
	}()

	payload := make([]byte, *payloadBytes)

	if *reps == 0 {
		infiniteSendLoop(&done, c, &interrupt, &payload)
	} else {
		sendNTimes(*reps, c, &done, &payload)
	}
}

func sendNTimes(n int, c *websocket.Conn, done *chan struct{}, payload *[]byte) {
	for i := 0; i < n; i++ {
		timestamp := getTimestamp()
		jsonMap := DataJSON{Timestamp: timestamp, Payload: *payload}
		marshal, _ := json.Marshal(jsonMap)
		err := c.WriteMessage(websocket.TextMessage, marshal)
		if err != nil {
			log.Println("write: ", err)
			return
		}
		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
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

func infiniteSendLoop(done *chan struct{}, c *websocket.Conn, interrupt *chan os.Signal, payload *[]byte) {

	ticker := time.NewTicker(time.Duration(*interval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-*done:
			return
		case <-ticker.C:
			timestamp := getTimestamp()
			jsonMap := DataJSON{ Timestamp: timestamp, Payload: *payload}
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
