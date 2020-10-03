package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

func readDispatcher(decoder *json.Decoder,
										done *chan struct{},
										toolRtt *os.File) {
	defer close(*done)
	defer toolRtt.Close()

	var wgReader sync.WaitGroup
	var mux sync.Mutex

	for {
		var message DataJSON
		err := decoder.Decode(&message)
		if err != nil {
			log.Println("read: ", err)
			return
		}
		if message.Id == 0 {
			log.Println("Closing socket")
			wgReader.Wait()
			return
		}
		wgReader.Add(1)

		// dispatch read
		go singleRead(&wgReader, &message, &mux, toolRtt)
	}
}

func singleRead(wgReader *sync.WaitGroup,
								jsonMap *DataJSON,
								mux *sync.Mutex,
								toolRtt *os.File) {
	defer wgReader.Done()
	latency := getTimestamp().Sub(jsonMap.ClientTimestamp)
	log.Printf("%d.\t%d.%d ms", jsonMap.Id, latency.Milliseconds(), latency%time.Millisecond)
	mux.Lock()
	toolRtt.WriteString(strconv.FormatInt(latency.Milliseconds(), 10) + "." + strconv.Itoa(int(latency%time.Millisecond)))
	toolRtt.WriteString(",")
	toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.UnixNano(), 10))
	toolRtt.WriteString(",")
	toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.UnixNano(), 10))
	toolRtt.WriteString("\n")
	mux.Unlock()
}
