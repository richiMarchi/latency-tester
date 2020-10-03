package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

func sendNTimes(n uint64,
								encoder *json.Encoder,
								interrupt *chan os.Signal,
								payload *string,
								ssReading *bool,
								ssHandling *chan bool) {
	var i uint64
	for i = 1; i < n; i++ {
		jsonMap := DataJSON{Id: i, Payload: *payload, ClientTimestamp: getTimestamp(), ServerTimestamp: time.Time{}}
		*ssReading = true
		*ssHandling <- true
		err := encoder.Encode(&jsonMap)
		*ssReading = false
		if err != nil {
			log.Println("write: ", err)
			gracefulSocketClose(encoder)
			return
		}
		select {
		case <-*interrupt:
			log.Println("interrupt")
			gracefulSocketClose(encoder)
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
	jsonMap := DataJSON{Id: 0}
	err := encoder.Encode(&jsonMap)
	if err != nil {
		log.Println("write: ", err)
		return
	}
}

func infiniteSendLoop(encoder *json.Encoder,
											interrupt *chan os.Signal,
											payload *string,
											ssReading *bool,
											ssHandling *chan bool) {

	ticker := time.NewTicker(time.Duration(*interval) * time.Millisecond)
	defer ticker.Stop()

	var id uint64 = 1
	for {
		select {
		case <-ticker.C:
			jsonMap := DataJSON{ Id: id, Payload: *payload, ClientTimestamp: getTimestamp(), ServerTimestamp: time.Time{}}
			*ssReading = true
			*ssHandling <- true
			err := encoder.Encode(&jsonMap)
			*ssReading = false
			id = id + 1
			if err != nil {
				log.Println("write: ", err)
				gracefulSocketClose(encoder)
				return
			}
		case <-*interrupt:
			log.Println("interrupt")
			gracefulSocketClose(encoder)
			return
		}
	}
}

func gracefulSocketClose(encoder *json.Encoder) {
	jsonMap := DataJSON{Id: 0}
	err := encoder.Encode(&jsonMap)
	if err != nil {
		log.Println("write: ", err)
		return
	}
}
