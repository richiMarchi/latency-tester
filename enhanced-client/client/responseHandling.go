package main

import (
	"encoding/json"
	"fmt"
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
				fmt.Println("read: ", err)
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
		fmt.Printf("%d.\t%f ms\n", jsonMap.Id, float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()))
		mux.Lock()
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatFloat(
			float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()), 'f', -1, 64))
		toolRtt.WriteString("\n")
	}
}
