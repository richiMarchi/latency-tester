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
