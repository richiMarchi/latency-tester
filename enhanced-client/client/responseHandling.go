package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

func readDispatcher(
	c *websocket.Conn,
	wg *sync.WaitGroup,
	toolRtt *os.File) {
	var mux sync.Mutex
	_, message, _ := c.ReadMessage()
	singleRead(&message, &mux, toolRtt)
	wg.Done()
}

func singleRead(
	message *[]byte,
	mux *sync.Mutex,
	toolRtt *os.File) {
	defer mux.Unlock()
	var jsonMap DataJSON
	_ = json.Unmarshal(*message, &jsonMap)
	latency := getTimestamp().Sub(jsonMap.ClientTimestamp)
	if jsonMap.Id == 0 {
		log.Println("Connection Reset")
		mux.Lock()
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",-1\n")
	} else {
		log.Printf("%d.\t%d.%d ms", jsonMap.Id, latency.Milliseconds(), latency%time.Millisecond)
		mux.Lock()
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(latency.Milliseconds(), 10) + "." +
			strconv.Itoa(int(latency%time.Millisecond)))
		toolRtt.WriteString("\n")
	}
}
