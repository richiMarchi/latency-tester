package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strconv"
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
			log.Println("read: ", err)
			return
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
	var jsonMap DataJSON
	_ = json.Unmarshal(*message, &jsonMap)
	latency := getTimestamp().Sub(jsonMap.ClientTimestamp)
	log.Printf("%d.\t%d.%d ms", jsonMap.Id+1, latency.Milliseconds(), latency%time.Millisecond)
	mux.Lock()
	toolRtt.WriteString(strconv.FormatInt(latency.Milliseconds(), 10) + "." + strconv.Itoa(int(latency%time.Millisecond)))
	toolRtt.WriteString(",")
	toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.UnixNano(), 10))
	toolRtt.WriteString(",")
	toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.UnixNano(), 10))
	toolRtt.WriteString("\n")
	mux.Unlock()
}
