package main

import (
	"encoding/json"
	"flag"
	"log"
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
	decoder := json.NewDecoder(r.Body)
	var jsonMap DataJSON
	err := decoder.Decode(&jsonMap)
	if err != nil {
		log.Println("read: ", err)
		log.Println()
		return
	}
	log.Printf("recv: ACK")
	jsonMap.ServerTimestamp = getTimestamp()
	jsonMap.Payload = randomString(uint(jsonMap.ResponseSize) - 62 /* offset to set the perfect desired message size */)
	encoder := json.NewEncoder(w)
	err = encoder.Encode(jsonMap)
	if err != nil {
		log.Println("write: ", err)
		return
	}
}
