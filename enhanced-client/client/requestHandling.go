package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

func requestSender(
	interrupt chan os.Signal,
	msgId *uint64,
	toolRtt *os.File) {
	payload := randomString(*requestBytes - 62 /* offset to set the perfect desired message size */)
	// If *reps == 0 then loop infinitely, otherwise loop *reps times
	if *reps != 0 {
		*reps += 1
	}
	var readersWg sync.WaitGroup
	for *msgId = 1; *msgId != *reps; *msgId++ {
		tmp := getTimestamp()
		jsonMap := DataJSON{
			Id:              *msgId,
			Payload:         payload,
			ClientTimestamp: tmp,
			ServerTimestamp: time.Time{},
			ResponseSize:    *responseBytes,
		}
		// Parallel read dispatcher
		readersWg.Add(1)
		marshal, _ := json.Marshal(jsonMap)
		resp, err := http.Post(address, "application/json", bytes.NewBuffer(marshal))
		if err != nil {
			log.Printf("Error sending message %d: %s", *msgId, err.Error())
		} else {
			go readDispatcher(resp, &readersWg, toolRtt)
		}
		tsDiff := (time.Duration(*interval) * time.Millisecond) - time.Duration(getTimestamp().Sub(tmp).Nanoseconds())
		if tsDiff < 0 {
			tsDiff = 0
			log.Println("Warning: It was not possible to send message", *msgId+1, "after the desired interval!")
		}
		select {
		case <-interrupt:
			log.Println("interrupt")
			return
		case <-time.After(tsDiff):
		}
	}
	readersWg.Wait()
}
