package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"sync/atomic"
	"time"
)

func sendNTimes(n uint64,
	c *websocket.Conn,
	interrupt *chan os.Signal,
	payload *string,
	ssReading *bool,
	ssHandling *chan bool,
	networkPackets *uint64,
	reset *chan bool) {
	var i uint64
	errorTry := 0
	for i = 1; i <= n; i++ {
		tmp := getTimestamp()
		jsonMap := DataJSON{Id: i, Payload: *payload, ClientTimestamp: tmp, ServerTimestamp: time.Time{}}
		marshal, _ := json.Marshal(jsonMap)
		atomic.AddUint64(networkPackets, 1)
		*ssReading = true
		*ssHandling <- true
		err := c.WriteMessage(websocket.TextMessage, marshal)
		*ssReading = false
		for err != nil {
			if errorTry == errorRetry {
				log.Println("Could not reset the connection.")
				return
			}
			log.Printf("Trying to reset connection: %d/%d\n", errorTry, errorRetry)
			c = connect()
			jsonMap.Id = 0
			jsonMap.Payload = "Connection Reset"
			resetMarshal, _ := json.Marshal(jsonMap)
			err = c.WriteMessage(websocket.TextMessage, resetMarshal)
			*reset <- true
			errorTry += 1
		}
		errorTry = 0
		tsDiff := (time.Duration(*interval) * time.Millisecond) - time.Duration(getTimestamp().Sub(tmp).Nanoseconds())
		if tsDiff < 0 {
			tsDiff = 0
			log.Println("Warning: It was not possible to send message", i+1, "after the desired interval!")
		}
		select {
		case <-*interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			return
		case <-time.After(tsDiff):
		}
	}
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close: ", err)
		return
	}
}

func infiniteSendLoop(c *websocket.Conn,
	interrupt *chan os.Signal,
	payload *string,
	ssReading *bool,
	ssHandling *chan bool,
	networkPackets *uint64,
	reset *chan bool) {
	var id uint64 = 1
	errorTry := 0
	for {
		tmp := getTimestamp()
		jsonMap := DataJSON{Id: id, Payload: *payload, ClientTimestamp: tmp, ServerTimestamp: time.Time{}}
		marshal, _ := json.Marshal(jsonMap)
		atomic.AddUint64(networkPackets, 1)
		*ssReading = true
		*ssHandling <- true
		err := c.WriteMessage(websocket.TextMessage, marshal)
		*ssReading = false
		id = id + 1
		for err != nil {
			if errorTry == errorRetry {
				log.Println("Could not reset the connection.")
				atomic.StoreUint64(networkPackets, 0)
				return
			}
			log.Printf("Trying to reset connection: %d/%d\n", errorTry+1, errorRetry)
			c = connect()
			jsonMap.Id = 0
			jsonMap.Payload = "Connection Reset"
			resetMarshal, _ := json.Marshal(jsonMap)
			err = c.WriteMessage(websocket.TextMessage, resetMarshal)
			*reset <- true
			errorTry += 1
		}
		errorTry = 0
		tsDiff := (time.Duration(*interval) * time.Millisecond) - time.Duration(getTimestamp().Sub(tmp).Nanoseconds())
		if tsDiff < 0 {
			tsDiff = 0
			log.Println("Warning: It was not possible to send message", id, "after the desired interval!")
		}
		select {
		case <-*interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			return
		case <-time.After(tsDiff):
		}
	}
}
