package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type DataJSON struct {
	Id      uint64
	ServerTimestamp time.Time
	Payload string
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

var upgrader = websocket.Upgrader{}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
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

	log.Println("Connection established with ", r.Host)
	log.Println("Response payload size = ", responseBytes)

	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read: ", err)
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

func getTimestamp() time.Time {
	return time.Now()
}

func stringWithCharset(length uint, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length uint) string {
	return stringWithCharset(length, charset)
}
