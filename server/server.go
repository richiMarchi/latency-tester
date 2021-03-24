package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"github.com/richiMarchi/latency-tester/server/serialization/protobuf"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net/http"
	"strconv"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")
var tls = flag.Bool("tls", false, "true if tls server")

var upgrader = websocket.Upgrader{}

func main() {
	flag.Parse()
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { return })
	log.Println("Listening to", *addr)
	log.Println("TLS enabled:", *tls)
	if *tls {
		log.Fatal(http.ListenAndServeTLS(*addr, "server.crt", "server.key", nil))
	} else {
		log.Fatal(http.ListenAndServe(*addr, nil))
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade: ", err)
		return
	}
	_, msg, resErr := c.ReadMessage()
	if resErr != nil {
		log.Println("read: ", resErr)
		return
	}
	responseBytes, _ := strconv.Atoi(string(msg))

	payload := make([]byte, responseBytes)

	printLogs(c.RemoteAddr(), responseBytes)

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read: " + err.Error() + "\n")
			return
		}
		jsonMap := &protobuf.DataJSON{}
		_ = proto.Unmarshal(message, jsonMap)
		jsonMap.ServerTimestamp = timestamppb.New(getTimestamp())
		jsonMap.Payload = payload
		message, _ = proto.Marshal(jsonMap)
		err = c.WriteMessage(mt, message)
		log.Printf("recv: ACK")
		if err != nil {
			log.Println("write: ", err)
			return
		}
	}
}
