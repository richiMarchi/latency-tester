package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"time"
)

func requestSender(
	c *websocket.Conn,
	interrupt chan os.Signal,
	ssReading *bool,
	ssHandling chan uint64,
	reset chan *websocket.Conn) {
	payload := randomString(*requestBytes - 62 /* offset to set the perfect desired message size */)
	var id uint64
	// If *reps == 0 then loop infinitely, otherwise loop *reps times
	if *reps != 0 {
		*reps += 1
	}
	for id = 1; id != *reps; id++ {
		tmp := getTimestamp()
		jsonMap := DataJSON{
			Id:              id,
			Payload:         payload,
			ClientTimestamp: tmp,
			ServerTimestamp: time.Time{},
		}
		marshal, _ := json.Marshal(jsonMap)
		*ssReading = true
		ssHandling <- id
		err := c.WriteMessage(websocket.TextMessage, marshal)
		*ssReading = false
		for err != nil {
			log.Printf("Trying to reset connection...")
			c = connect()
			reset <- c
			reset <- c
			jsonMap.Id = 0
			jsonMap.Payload = "Connection Reset"
			resetMarshal, _ := json.Marshal(jsonMap)
			err = c.WriteMessage(websocket.TextMessage, resetMarshal)
		}
		tsDiff := (time.Duration(*interval) * time.Millisecond) - time.Duration(getTimestamp().Sub(tmp).Nanoseconds())
		if tsDiff < 0 {
			tsDiff = 0
			log.Println("Warning: It was not possible to send message", id+1, "after the desired interval!")
		}
		select {
		case <-interrupt:
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
