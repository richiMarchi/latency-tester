package main

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/richiMarchi/latency-tester/enhanced-client/client/serialization/protobuf"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func postAndRead(marshal *[]byte, toolRtt *os.File) {
	resp, err := http.Post(address, "application/json", bytes.NewBuffer(*marshal))
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
	jsonMap := &protobuf.DataJSON{}
	_ = proto.Unmarshal(message, jsonMap)
	singleRead(jsonMap, toolRtt)
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
