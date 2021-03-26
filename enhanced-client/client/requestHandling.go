package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

func requestSender(interrupt chan os.Signal, msgId *int32, toolRtt *os.File) {
	payload := make([]byte, *requestBytes)
	rand.Read(payload)
	// If *reps == 0 then loop infinitely, otherwise loop *reps times
	if *reps != 0 {
		*reps += 1
	}

	for *msgId = 1; *msgId != int32(*reps); *msgId++ {
		tmp := getTimestamp()
		go postAndRead(*msgId, &payload, toolRtt)

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
}
