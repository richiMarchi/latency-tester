package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")
var interval = flag.Int("interval", 0, "response interval time (ms)")

var upgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade: ", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read: ", err)
			break
		}
		time.Sleep(time.Duration(*interval) * time.Millisecond)
		err = c.WriteMessage(mt, message)
		log.Printf("recv: ACK")
		if err != nil {
			log.Println("write: ", err)
			break
		}
 	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}