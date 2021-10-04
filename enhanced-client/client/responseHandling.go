package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/richiMarchi/latency-tester/enhanced-client/client/serialization/protobuf"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func postAndRead(msgId int32, payload *[]byte, toolRtt *os.File) {
	client := &http.Client{Transport: &transport}
	reqMsg := &protobuf.DataJSON{
		Id:              msgId,
		Payload:         *payload,
		ClientTimestamp: timestamppb.New(getTimestamp()),
		ServerTimestamp: &timestamp.Timestamp{},
		ResponseSize:    int32(*responseBytes),
	}
	// Parallel read dispatcher
	marshal, _ := proto.Marshal(reqMsg)
	resp, err := client.Post(address, "application/json", bytes.NewBuffer(marshal))
	if err != nil {
		log.Printf("Error sending message: %s", err.Error())
		return
	}
	defer resp.Body.Close()
	message, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("read: ", err)
		log.Println()
		return
	}
	respMsg := &protobuf.DataJSON{}
	_ = proto.Unmarshal(message, respMsg)
	singleRead(respMsg, toolRtt)
}

func singleRead(
	jsonMap *protobuf.DataJSON, toolRtt *os.File) {
	if jsonMap.Id == 0 {
		log.Println("Connection Reset")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",-1\n")
	} else {
		latency := getTimestamp().Sub(jsonMap.ClientTimestamp.AsTime())
		log.Printf("%d.\t%f ms\n", jsonMap.Id, float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()))
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ClientTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatInt(jsonMap.ServerTimestamp.AsTime().UnixNano(), 10))
		toolRtt.WriteString(",")
		toolRtt.WriteString(strconv.FormatFloat(
			float64(latency.Nanoseconds())/float64(time.Millisecond.Nanoseconds()), 'f', -1, 64))
		toolRtt.WriteString("\n")
	}
}
