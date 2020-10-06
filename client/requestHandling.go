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
	networkPackets *uint64) {
	var i uint64
	for i = 0; i < n; i++ {
		tmp := getTimestamp()
		jsonMap := DataJSON{Id: i, Payload: *payload, ClientTimestamp: tmp, ServerTimestamp: time.Time{}}
		marshal, _ := json.Marshal(jsonMap)
		atomic.AddUint64(networkPackets, 1)
		*ssReading = true
		*ssHandling <- true
		err := c.WriteMessage(websocket.TextMessage, marshal)
		*ssReading = false
		if err != nil {
			log.Println("write: ", err)
			return
		}
		tsDiff := (time.Duration(*interval) * time.Millisecond) - time.Duration(getTimestamp().Sub(tmp).Nanoseconds())
		if tsDiff < 0 {
			tsDiff = 0
			log.Println("Warning: It was not possible to send message", i + 1 , "after the desired interval!")
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
	networkPackets *uint64) {

	var id uint64 = 0
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
		if err != nil {
			log.Println("write: ", err)
			return
		}
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
