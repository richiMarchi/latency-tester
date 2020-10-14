package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func readDispatcher(
	c *websocket.Conn,
	done chan struct{},
	toolRtt *os.File,
	reset chan *websocket.Conn) {
	var wgReader sync.WaitGroup
	var mux sync.Mutex
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "1000") {
				log.Println("read: ", err)
				wgReader.Wait()
				close(done)
				return
			} else {
				log.Println("Reader thread: waiting for connection to reset...")
				c = <-reset
				log.Println("Reader thread: connection reset signaled")
				continue
			}
		}
		wgReader.Add(1)

		// dispatch read
		go singleRead(&wgReader, &message, &mux, toolRtt)
	}
}

func singleRead(
	wgReader *sync.WaitGroup,
	message *[]byte,
	mux *sync.Mutex,
	toolRtt *os.File) {
	defer wgReader.Done()
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
