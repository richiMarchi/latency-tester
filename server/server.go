package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Error in net.Listen: %s", err)
	}
	for {
		conn, connErr := ln.Accept()
		if connErr != nil {
			log.Fatalf("Error in Accept() method: %s", connErr)
		}
		handleConnection(conn)
	}
}

func handleConnection(c net.Conn) {
	decoder := json.NewDecoder(c)
	encoder := json.NewEncoder(c)

	var responseBytes uint
	resErr := decoder.Decode(&responseBytes)
	if resErr != nil {
		log.Println("read: ", resErr)
		return
	}

	payload := randomString(responseBytes - 62 /* offset to set the perfect desired message size */)

	printLogs(c.RemoteAddr(), responseBytes)

	defer c.Close()
	for {
		var jsonMap DataJSON
		err := decoder.Decode(&jsonMap)
		if err != nil {
			log.Println("read: ", err)
			log.Println()
			return
		}
		jsonMap.ServerTimestamp = getTimestamp()
		jsonMap.Payload = payload
		err = encoder.Encode(&jsonMap)
		log.Printf("recv: ACK")
		if err != nil {
			log.Println("write: ", err)
			return
		}
	}
}
