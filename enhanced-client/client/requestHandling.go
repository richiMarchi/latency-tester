package main

import (
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/richiMarchi/latency-tester/enhanced-client/client/serialization/protobuf"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		jsonMap := &protobuf.DataJSON{
			Id:              *msgId,
			Payload:         payload,
			ClientTimestamp: timestamppb.New(tmp),
			ServerTimestamp: &timestamp.Timestamp{},
			ResponseSize:    int32(*responseBytes),
		}
		// Parallel read dispatcher
		marshal, _ := proto.Marshal(jsonMap)
		postAndRead(&marshal, toolRtt)

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
