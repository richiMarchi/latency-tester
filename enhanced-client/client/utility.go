package main

import (
	"crypto/tls"
	"fmt"
	"github.com/brucespang/go-tcpinfo"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type TimedTCPInfo struct {
	MsgId     int32
	Timestamp time.Time
	TcpInfo   *tcpinfo.TCPInfo
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func connect() *websocket.Conn {
	addrParts := strings.Split(address, "/")
	pathString := ""
	for _, part := range addrParts[1:] {
		pathString += "/" + part
	}
	var conn *websocket.Conn
	if *https {
		conf := &tls.Config{InsecureSkipVerify: true}
		dialer := websocket.Dialer{
			TLSClientConfig:  conf,
			HandshakeTimeout: 10 * time.Second,
		}
		if *srcPort != 0 {
			dialer.NetDialContext = (&net.Dialer{LocalAddr: &net.TCPAddr{Port: *srcPort}}).DialContext
		}
		u := url.URL{Scheme: "wss", Host: addrParts[0], Path: pathString + "/echo"}
		c, _, err := dialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial: ", err)
		}
		conn = c
	} else {
		dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
		if *srcPort != 0 {
			dialer.NetDialContext = (&net.Dialer{LocalAddr: &net.TCPAddr{Port: *srcPort}}).DialContext
		}
		u := url.URL{Scheme: "ws", Host: addrParts[0], Path: "/echo"}
		c, _, err := dialer.Dial(u.String(), nil)
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
	fmt.Println("Repetitions:\t\t", *reps)
	fmt.Println("Request Bytes:\t\t", *requestBytes)
	fmt.Println("Response Bytes:\t\t", *responseBytes)
	fmt.Println("Send Interval:\t\t", *interval)
	fmt.Println("TLS enabled:\t\t", *https)
	fmt.Println("Traceroute IP:\t", *tracerouteIp)
	fmt.Println("TCP Stats enabled:\t", *sockOpt)
	fmt.Println("Address:\t\t", address)
	fmt.Println()
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
