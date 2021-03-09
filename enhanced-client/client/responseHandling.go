package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func readDispatcher(
	resp *http.Response,
	wg *sync.WaitGroup,
	toolRtt *os.File) {
	defer resp.Body.Close()
	var mux sync.Mutex
	decoder := json.NewDecoder(resp.Body)
	var jsonMap DataJSON
	err := decoder.Decode(&jsonMap)
	if err != nil {
		log.Println("read: ", err)
		log.Println()
		return
	}
	singleRead(jsonMap, &mux, toolRtt)
	wg.Done()
}

func singleRead(
	jsonMap DataJSON,
	mux *sync.Mutex,
	toolRtt *os.File) {
	defer mux.Unlock()
	latency := getTimestamp().Sub(jsonMap.ClientTimestamp)
	if jsonMap.Id == 0 {
		log.Println("Connection Reset")
		mux.Lock()
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.UnixNano(), 10))
		toolRtt.WriteString(",-1\n")
	} else {
		log.Printf("%d.\t%f ms\n", jsonMap.Id, float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()))
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
