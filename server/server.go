package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
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

	payload := randomString(uint(responseBytes) - 62 /* offset to set the perfect desired message size */)

	printLogs(c.RemoteAddr(), responseBytes)

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read: ", err)
			log.Println()
			return
		}
		var jsonMap DataJSON
		_ = json.Unmarshal(message, &jsonMap)
		jsonMap.ServerTimestamp = getTimestamp()
		jsonMap.Payload = payload
		message, _ = json.Marshal(jsonMap)
		err = c.WriteMessage(mt, message)
		log.Printf("recv: ACK")
		if err != nil {
			log.Println("write: ", err)
			return
		}
	}
}
