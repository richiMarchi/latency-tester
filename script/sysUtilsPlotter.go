package main

import (
	"bufio"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func PingPlotter(destination string) {
	fmt.Println("Plotting Ping")
	p, err := plot.New()
	errMgmt(err)

	p.X.Label.Text = "Time (s)"
	p.Y.Label.Text = "OS RTT (ms)"
	p.Title.Text = "Ping destination: " + destination

	// Open the desired files
	file, err := os.Open("/tmp/ping_report.txt")
	if err != nil {
		log.Fatal(err)
	}

	var values plotter.XYs
	var firstTs float64
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "time=") && strings.Contains(line, " ms") {
			lineTs := line[1:strings.Index(line, ".")]
			floatMs := line[strings.Index(line, "time=")+5 : strings.Index(line, " ms")]
			timeInter, err := strconv.ParseFloat(lineTs, 64)
			errMgmt(err)
			rttVal, err := strconv.ParseFloat(floatMs, 64)
			if len(values) == 0 {
				firstTs = timeInter
			}
			values = append(values, plotter.XY{X: timeInter - firstTs, Y: rttVal})
		}
	}
	err = plotutil.AddLines(p, "Ping RTT", values)

	if err := p.Save(1000, 1000, "/tmp/pingPlot.png"); err != nil {
		panic(err)
	}
}

func TCPdumpPlotter() {
	fmt.Println("Plotting TCP RTT")
	p, err := plot.New()
	errMgmt(err)

	p.X.Label.Text = "Packets"
	p.Y.Label.Text = "TCP RTT (ms)"
	p.Title.Text = "TCP Latency"

	var values plotter.XYs
	var counter float64 = 0
	// Open the desired files
	if handle, err := pcap.OpenOffline("/tmp/tcpdump_report.pcap"); err != nil {
		panic(err)
	} else {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		var tmpPackets []gopacket.Packet
		for packet := range packetSource.Packets() {
			if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
				tcp, _ := tcpLayer.(*layers.TCP)
				if tcp.ACK {
					seqPacket := ackedPacketInSlice(tcp.Ack, &tmpPackets)
					if seqPacket != nil {
						values = append(values, plotter.XY{X: counter, Y: float64(packet.Metadata().Timestamp.Sub(seqPacket.Metadata().Timestamp).Milliseconds())})
						counter += 1
					}
				}
				tmpPackets = append(tmpPackets, packet)
			}
		}
	}
	err = plotutil.AddLines(p, "TCP RTT", values)

	if err := p.Save(1000, 1000, "/tmp/tcpPlot.png"); err != nil {
		panic(err)
	}
}
