package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"time"
)

func sendNTimes(n uint64,
		c *websocket.Conn,
		done *chan struct{},
		interrupt *chan os.Signal,
		payload *string,
		tsMap *map[uint64]time.Time) {
	var i uint64
	for i = 0; i < n; i++ {
		jsonMap := DataJSON{Id: i, Payload: *payload, ServerTimestamp: time.Time{}}
		marshal, _ := json.Marshal(jsonMap)
		err := c.WriteMessage(websocket.TextMessage, marshal)
		(*tsMap)[i] = getTimestamp()
		if err != nil {
			log.Println("write: ", err)
			return
		}
		select {
		case <-*interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			select {
			case <-*done:
			case <-time.After(time.Second):
			}
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close: ", err)
		return
	}
}

func infiniteSendLoop(done *chan struct{},
		c *websocket.Conn,
		interrupt *chan os.Signal,
		payload *string,
		tsMap *map[uint64]time.Time) {

	ticker := time.NewTicker(time.Duration(*interval) * time.Millisecond)
	defer ticker.Stop()

	var id uint64
	id = 0

	for {
		select {
		case <-*done:
			return
		case <-ticker.C:
			jsonMap := DataJSON{ Id: id, Payload: *payload, ServerTimestamp: time.Time{}}
			marshal, _ := json.Marshal(jsonMap)
			err := c.WriteMessage(websocket.TextMessage, marshal)
			(*tsMap)[id] = getTimestamp()
			id = id + 1
			if err != nil {
				log.Println("write: ", err)
				return
			}
		case <-*interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			select {
			case <-*done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}