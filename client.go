package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run client.go <host> <port>")
		os.Exit(1)
	}
	address := os.Args[1]
	port := os.Args[2]

	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: address + ":" + port, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	done := make(chan struct{})

	// Parallel read loop
	go func() {
		defer close(done)
		for  {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read: ", err)
				return
			}
			oldTs, _ := strconv.ParseInt(string(message), 10, 64)
			latency := getTimestamp() - oldTs
			log.Printf("recv: %d.%d ms", latency / int64(time.Millisecond), latency % int64(time.Millisecond))
		}
	}()

	ticker := time.NewTicker(time.Second)
	var timestamp int64
	defer ticker.Stop()

	// Main send loop
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			timestamp = getTimestamp()
			err := c.WriteMessage(websocket.TextMessage, []byte(strconv.FormatInt(timestamp, 10)))
			if err != nil {
				log.Println("write: ", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close: ", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func getTimestamp() int64 {
	return time.Now().UnixNano()
}
