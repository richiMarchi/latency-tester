package main

import (
	"fmt"
	"github.com/brucespang/go-tcpinfo"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

func customTraceroute(
	tracerouteIp string,
	outputFile *os.File) {
	output, _ := exec.Command("traceroute", tracerouteIp).Output()
	outputFile.WriteString(string(output))
}

func getSocketStats(
	conn *websocket.Conn,
	ssReading *bool,
	outputFile *os.File,
	wg *sync.WaitGroup,
	msgId *uint64,
	reset chan *websocket.Conn) {
	defer wg.Done()
	defer outputFile.Close()

	tcpConn := getTCPConnFromWebsocketConn(conn)
	var sockOpt []TimedTCPInfo
	for *ssReading {
		// Check if the connection changed
		select {
		case conn = <-reset:
			tcpConn = getTCPConnFromWebsocketConn(conn)
			outputFile.WriteString(strconv.FormatInt(getTimestamp().UnixNano(), 10) + ",-1,Connection Reset\n")
		default:
		}
		if *msgId != 0 {
			tcpInfo, _ := tcpinfo.GetsockoptTCPInfo(tcpConn)
			sockOpt = append(sockOpt, TimedTCPInfo{
				MsgId:     *msgId,
				Timestamp: getTimestamp(),
				TcpInfo:   tcpInfo,
			})
		}
	}
	log.Println("Saving TCP Stats to file...")
	for i, info := range sockOpt {
		if i == 0 || !cmp.Equal(sockOpt[i].MsgId, sockOpt[i-1].MsgId) ||
			!cmp.Equal(sockOpt[i].TcpInfo, sockOpt[i-1].TcpInfo) {
			str := fmt.Sprintf("%v", *info.TcpInfo)
			str = strings.ReplaceAll(str[1:len(str)-1], " ", ",")
			str = strings.ReplaceAll(str, "[", "")
			str = strings.ReplaceAll(str, "]", "")
			outputFile.WriteString(strconv.FormatInt(info.Timestamp.UnixNano(), 10) + "," +
				strconv.FormatUint(info.MsgId, 10) + "," + str + "\n")
		}
	}
}
