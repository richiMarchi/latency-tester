package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/richiMarchi/latency-tester/enhanced-client/client/serialization/protobuf"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func readDispatcher(
	c *websocket.Conn,
	done chan struct{},
	toolRtt *os.File,
	reset chan *websocket.Conn) {
	for {
		// Read all incoming messages
		_, message, err := c.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "1000") {
				fmt.Println("read: ", err)
				close(done)
				return
			} else {
				log.Println("Reader thread: waiting for connection to reset...")
				c = <-reset
				log.Println("Reader thread: connection reset signaled")
				continue
			}
		}

		handleMessage(&message, toolRtt)
	}
}

// Deserialize the message received and store data in the file
func handleMessage(message *[]byte, toolRtt *os.File) {
	jsonMap := &protobuf.DataJSON{}
	_ = proto.Unmarshal(*message, jsonMap)
	if jsonMap.Id == 0 {
		log.Println("Connection Reset")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",-1\n")
	} else {
		latency := getTimestamp().Sub(jsonMap.ClientTimestamp.AsTime())
		fmt.Printf("%d.\t%f ms\n", jsonMap.Id, float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()))
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatFloat(
			float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()), 'f', -1, 64))
		toolRtt.WriteString("\n")
	}
}
