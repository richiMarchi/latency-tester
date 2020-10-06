package main

import (
	"crypto/tls"
	"log"
	"math/rand"
	"net"
	"reflect"
	"time"
	"unsafe"
)

type DataJSON struct {
	Id              uint64
	ClientTimestamp time.Time
	ServerTimestamp time.Time
	Payload         string
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func getTimestamp() time.Time {
	return time.Now()
}

func stringWithCharset(length uint64, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length uint64) string {
	return stringWithCharset(length, charset)
}

func printLogs(reps uint64,
	requestBytes uint64,
	responseBytes uint64,
	interval uint64,
	tls bool,
	traceroute bool,
	address string,
	pingIp string) {
	log.Println("Repetitions:\t\t", reps)
	log.Println("Request Bytes:\t\t", requestBytes)
	log.Println("Response Bytes:\t\t", responseBytes)
	log.Println("Send Interval:\t\t", interval)
	log.Println("TLS enabled:\t\t", tls)
	log.Println("Traceroute enabled:\t", traceroute)
	log.Println("Address:\t\t", address)
	log.Println("Ping and Traceroute IP:\t", pingIp)
	log.Println()
}

// to get the internal wrapped connection from the tls.Conn
func getConnFromTLSConn(tlsConn *tls.Conn) net.Conn {
	// awful workaround until https://github.com/golang/go/issues/29257 is solved.
	conn := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
	conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
	return conn.Interface().(net.Conn)
}
