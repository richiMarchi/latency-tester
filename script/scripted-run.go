package main

import (
	"github.com/go-ping/ping"
	"github.com/lorenzosaino/go-sysctl"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

func main() {
	// Print flags statuses in order to be sure it is as expected
	val, err := sysctl.Get("net.ipv4.tcp_slow_start_after_idle")
	errMgmt(err)
	log.Println("TCP slow start after idle value: ", val)
	val, err = sysctl.Get("net.ipv4.tcp_congestion_control")
	errMgmt(err)
	log.Println("TCP congestion control algorithm: ", val)

	// Start ping and tcpdump in background
	c := make(chan os.Signal, 2)
	var wg sync.WaitGroup
	//TODO: add parsed address to ping
	wg.Add(2)
	go pinger(&wg, "127.0.0.1", c)
	go tcpDumper(&wg, c)

	//TODO: loops

	c <- os.Interrupt
	c <- os.Interrupt
	wg.Wait()
}

func pinger(wg *sync.WaitGroup, address string, c chan os.Signal) {
	// Create output file
	osRtt, err := os.Create("/tmp/ping_report.csv")
	errMgmt(err)
	osRtt.WriteString("#timestamp,os-rtt\n")
	defer osRtt.Close()

	// Create pinger and set interval
	pinger, err := ping.NewPinger(address)
	errMgmt(err)
	pinger.Interval = 30 * time.Second

	// Handle stop
	go func() {
		for _ = range c {
			pinger.Stop()
		}
	}()

	// Handle packet reception
	pinger.OnRecv = func(pkt *ping.Packet) {
		osRtt.WriteString(strconv.FormatInt(getTimestamp().UnixNano(), 10) +
			"," + strconv.FormatInt(pkt.Rtt.Nanoseconds(), 10) + "\n")
	}

	err = pinger.Run()
	wg.Done()
}

func tcpDumper(wg *sync.WaitGroup, c chan os.Signal) {
	// Create output file
	tcpdumpFile, err := os.Create("/tmp/tcpdump_report.pcap")
	errMgmt(err)
	defer tcpdumpFile.Close()

	tcpdumper := exec.Command("tcpdump", "-U", "-s", "96", "-w", "/tmp/tcpdump_report.pcap")

	// Handle stop
	go func() {
		for _ = range c {
			_ = tcpdumper.Process.Signal(os.Interrupt)
		}
	}()

	err = tcpdumper.Run()
	wg.Done()
}

func getTimestamp() time.Time {
	return time.Now()
}

func errMgmt(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
