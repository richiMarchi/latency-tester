package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func readDispatcher(c *websocket.Conn,
	stop *bool,
	done *chan struct{},
	toolRtt *os.File,
	networkPackets *uint64) {
	defer close(*done)
	defer toolRtt.Close()

	var wgReader sync.WaitGroup
	var mux sync.Mutex
	for !*stop || atomic.LoadUint64(networkPackets) > 0 {
		_, message, err := c.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "1000") {
				log.Println("read: ", err)
				return
			} else {
				time.Sleep(1 * time.Second)
				continue
			}
		}
		wgReader.Add(1)

		// dispatch read
		go singleRead(&wgReader, &message, &mux, toolRtt)
		atomic.AddUint64(networkPackets, ^uint64(0))
	}
	wgReader.Wait()
}

func singleRead(wgReader *sync.WaitGroup,
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
		toolRtt.WriteString(strconv.FormatInt(latency.Milliseconds(), 10) + "." + strconv.Itoa(int(latency%time.Millisecond)))
		toolRtt.WriteString("\n")
	}
}
