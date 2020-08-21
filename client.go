package main

import (
	"encoding/csv"
	"encoding/json"
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

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: go run client.go <host:port> <repetitions> <log-file>")
		os.Exit(1)
	}
	address := os.Args[1]
	reps, convErr := strconv.Atoi(os.Args[2])
	if convErr != nil || reps < 0 {
		fmt.Fprintln(os.Stderr, "<repetitions> must be a positive number")
	}
	logFile := os.Args[3]

	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: address, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	done := make(chan struct{})

	results, err := os.Create(logFile)
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
			oldTs := jsonMap.Timestamp
			latency := getTimestamp() - oldTs
			log.Printf("recv: %d.%d ms", latency / int64(time.Millisecond), latency % int64(time.Millisecond))
			payloadReceived := jsonMap.Payload
			log.Println("PAYLOAD SIZE = ", len(payloadReceived))
			if !firstIteration {
				results.WriteString(",")
			} else {
				firstIteration = false
			}
			results.WriteString(strconv.Itoa(int(latency / int64(time.Millisecond))) + "." + strconv.Itoa(int(latency % int64(time.Millisecond))))
		}
	}()

	payload := make([]byte, 16)
	log.Println("STARTING PAYLOAD SIZE = ", len(payload))

	if reps == 0 {
		infiniteSendLoop(&done, c, &interrupt, &payload)
	} else {
		sendNTimes(reps, c, &done, &payload)
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
		time.Sleep(time.Second)
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

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-*done:
			return
		case <-ticker.C:
			timestamp := getTimestamp()
			jsonMap := map[string]int64{"interval": 500, "timestamp": timestamp}
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
