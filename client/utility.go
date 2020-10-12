package main

import (
	"crypto/tls"
	"github.com/brucespang/go-tcpinfo"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)

type DataJSON struct {
	Id              uint64
	ClientTimestamp time.Time
	ServerTimestamp time.Time
	Payload         string
}

type TimedTCPInfo struct {
	Timestamp time.Time
	TcpInfo   *tcpinfo.TCPInfo
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func connect() *websocket.Conn {
	var conn *websocket.Conn
	if *https {
		conf := &tls.Config{InsecureSkipVerify: true}
		dialer := websocket.Dialer{TLSClientConfig: conf}
		u := url.URL{Scheme: "wss", Host: address, Path: "/echo"}
		c, _, err := dialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial: ", err)
		}
		conn = c
	} else {
		u := url.URL{Scheme: "ws", Host: address, Path: "/echo"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial: ", err)
		}
		conn = c
	}
	_ = conn.WriteMessage(websocket.TextMessage, []byte(strconv.FormatUint(*responseBytes, 10)))
	return conn
}

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

func printLogs() {
	log.Println("Repetitions:\t\t", *reps)
	log.Println("Request Bytes:\t\t", *requestBytes)
	log.Println("Response Bytes:\t\t", *responseBytes)
	log.Println("Send Interval:\t\t", *interval)
	log.Println("TLS enabled:\t\t", *https)
	log.Println("Traceroute enabled:\t", *traceroute)
	log.Println("Address:\t\t", address)
	log.Println("Ping and Traceroute IP:\t", pingIp)
	log.Println()
}

func getTCPConnFromWebsocketConn(conn *websocket.Conn) *net.TCPConn {
	if *https {
		return getConnFromTLSConn(conn.UnderlyingConn().(*tls.Conn)).(*net.TCPConn)
	} else {
		return conn.UnderlyingConn().(*net.TCPConn)
	}
}

// to get the internal wrapped connection from the tls.Conn
func getConnFromTLSConn(tlsConn *tls.Conn) net.Conn {
	// awful workaround until https://github.com/golang/go/issues/29257 is solved.
	conn := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
	conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
	return conn.Interface().(net.Conn)
}
