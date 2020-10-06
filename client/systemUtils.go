package main

import (
	"fmt"
	"github.com/brucespang/go-tcpinfo"
	"github.com/google/go-cmp/cmp"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func customTraceroute(tracerouteIp string,
	outputFile *os.File) {
	defer outputFile.Close()
	output, _ := exec.Command("traceroute", tracerouteIp).Output()
	outputFile.WriteString(string(output))
}

func customPing(pingIp string,
	wGroup *sync.WaitGroup,
	done *chan struct{},
	outputFile *os.File) {
	defer wGroup.Done()
	defer outputFile.Close()
	for {
		output, _ := exec.Command("ping", pingIp, "-c 1").Output()
		rttMs := string(output)
		if strings.Contains(rttMs, "time=") && strings.Contains(rttMs, " ms") {
			floatMs := rttMs[strings.Index(rttMs, "time=")+5 : strings.Index(rttMs, " ms")]
			outputFile.WriteString(strconv.FormatInt(getTimestamp().UnixNano(), 10) + "," + floatMs + "\n")
		}
		select {
		case <-*done:
			return
		case <-time.After(time.Duration(*interval) * time.Millisecond):
		}
	}
}

func getSocketStats(conn *net.TCPConn,
	ssReading *bool,
	outputFile *os.File,
	wg *sync.WaitGroup,
	ssHandling *chan bool) {
	defer wg.Done()
	defer outputFile.Close()
	var sockOpt []*tcpinfo.TCPInfo
	var timestamps []time.Time
	msgId := 1
	for <-*ssHandling {
		for *ssReading {
			timestamps = append(timestamps, getTimestamp())
			tcpInfo, _ := tcpinfo.GetsockoptTCPInfo(conn)
			sockOpt = append(sockOpt, tcpInfo)
		}
		for i, info := range sockOpt {
			if i == 0 || !cmp.Equal(sockOpt[i], sockOpt[i-1]) {
				str := fmt.Sprintf("%v", *info)
				str = strings.ReplaceAll(str[1:len(str)-1], " ", ",")
				str = strings.ReplaceAll(str, "[", "")
				str = strings.ReplaceAll(str, "]", "")
				outputFile.WriteString(strconv.FormatInt(timestamps[i].UnixNano(), 10) + "," + strconv.Itoa(msgId) + "," + str + "\n")
			}
		}
		sockOpt = sockOpt[:0]
		msgId += 1
	}
}
