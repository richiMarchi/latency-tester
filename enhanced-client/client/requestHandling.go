package main

import (
	"fmt"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/gorilla/websocket"
	"github.com/richiMarchi/latency-tester/enhanced-client/client/serialization/protobuf"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"os"
	"time"
)

func requestSender(
	c *websocket.Conn,
	interrupt chan os.Signal,
	ssReading *bool,
	reset chan *websocket.Conn,
	msgId *int32) {
	payload := make([]byte, *requestBytes)
	// If *reps == 0 then loop infinitely, otherwise loop *reps times
	if *reps != 0 {
		*reps += 1
	}
	for *msgId = 1; *msgId != int32(*reps); *msgId++ {
		// Create the message with message ID and the current timestamp, serialize with protobuf and send it
		tmp := getTimestamp()
		jsonMap := &protobuf.DataJSON{
			Id:              *msgId,
			Payload:         payload,
			ClientTimestamp: timestamppb.New(tmp),
			ServerTimestamp: &timestamp.Timestamp{},
		}
		marshal, _ := proto.Marshal(jsonMap)
		err := c.WriteMessage(websocket.TextMessage, marshal)
		for err != nil {
			log.Printf("Trying to reset connection...")
			c = connect()
			reset <- c
			if *sockOpt {
				reset <- c
			}
			jsonMap.Id = 0
			jsonMap.Payload = []byte{}
			resetMarshal, _ := proto.Marshal(jsonMap)
			err = c.WriteMessage(websocket.TextMessage, resetMarshal)
		}
		tsDiff := (time.Duration(*interval) * time.Millisecond) - time.Duration(getTimestamp().Sub(tmp).Nanoseconds())
		if tsDiff < 0 {
			tsDiff = 0
			fmt.Println("WARNING: It was not possible to send message", *msgId+1, "after the desired interval!")
		}
		select {
		case <-interrupt:
			log.Println("interrupt")
			*ssReading = false
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			return
		case <-time.After(tsDiff):
		}
	}
	*ssReading = false
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close: ", err)
		return
	}
}
