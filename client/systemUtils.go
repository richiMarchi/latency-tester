package main

import (
	"fmt"
	"github.com/brucespang/go-tcpinfo"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func customTraceroute(
	tracerouteIp string,
	outputFile *os.File) {
	output, _ := exec.Command("traceroute", tracerouteIp).Output()
	outputFile.WriteString(string(output))
}

func customPing(
	pingIp string,
	wGroup *sync.WaitGroup,
	done chan struct{},
	outputFile *os.File) {
	defer wGroup.Done()
	for {
		output, _ := exec.Command("ping", pingIp, "-c 1").Output()
		rttMs := string(output)
		if strings.Contains(rttMs, "time=") && strings.Contains(rttMs, " ms") {
			floatMs := rttMs[strings.Index(rttMs, "time=")+5 : strings.Index(rttMs, " ms")]
			outputFile.WriteString(strconv.FormatInt(getTimestamp().UnixNano(), 10) + "," + floatMs + "\n")
		}
		select {
		case <-done:
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
}

func getSocketStats(
	conn *websocket.Conn,
	ssReading *bool,
	outputFile *os.File,
	wg *sync.WaitGroup,
	ssHandling chan uint64,
	reset chan *websocket.Conn) {
	defer wg.Done()

	tcpConn := getTCPConnFromWebsocketConn(conn)
	var sockOpt []TimedTCPInfo
	var msgId uint64
	for {
		msgId = <-ssHandling
		if msgId == 0 {
			break
		}
		for *ssReading {
			tcpInfo, _ := tcpinfo.GetsockoptTCPInfo(tcpConn)
			sockOpt = append(sockOpt, TimedTCPInfo{
				Timestamp: getTimestamp(),
				TcpInfo:   tcpInfo,
			})
		}
		for i, info := range sockOpt {
			if i == 0 || !cmp.Equal(sockOpt[i].TcpInfo, sockOpt[i-1].TcpInfo) {
				str := fmt.Sprintf("%v", *info.TcpInfo)
				str = strings.ReplaceAll(str[1:len(str)-1], " ", ",")
				str = strings.ReplaceAll(str, "[", "")
				str = strings.ReplaceAll(str, "]", "")
				outputFile.WriteString(strconv.FormatInt(info.Timestamp.UnixNano(), 10) + "," +
					strconv.FormatUint(msgId, 10) + "," + str + "\n")
			}
		}
		sockOpt = sockOpt[:0]
		select {
		case conn = <-reset:
			tcpConn = getTCPConnFromWebsocketConn(conn)
			outputFile.WriteString(strconv.FormatInt(getTimestamp().UnixNano(), 10) + ",-1,Connection reset\n")
		default:
		}
	}
}
