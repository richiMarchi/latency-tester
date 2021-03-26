package main

import (
	"flag"
	"github.com/golang/protobuf/proto"
	"github.com/richiMarchi/latency-tester/server/serialization/protobuf"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")
var tls = flag.Bool("tls", false, "true if tls server")

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { return })
	log.Println("Listening to", *addr, "\nTLS enabled:", *tls)
	if *tls {
		log.Fatal(http.ListenAndServeTLS(*addr, "server.crt", "server.key", nil))
	} else {
		log.Fatal(http.ListenAndServe(*addr, nil))
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	message, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("read: ", err)
		log.Println()
		return
	}
	jsonMap := &protobuf.DataJSON{}
	_ = proto.Unmarshal(message, jsonMap)
	log.Printf("recv: ACK")
	jsonMap.Payload = make([]byte, uint(jsonMap.ResponseSize))
	rand.Read(jsonMap.Payload)
	jsonMap.ServerTimestamp = timestamppb.New(getTimestamp())
	responseMsg, _ := proto.Marshal(jsonMap)
	_, err = w.Write(responseMsg)
	if err != nil {
		log.Println("write: ", err)
		return
	}
}
