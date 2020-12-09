package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"os"
	"os/exec"
	"sort"
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
	p.X.Tick.Marker = hplot.Ticks{N: 15}
	p.Y.Tick.Marker = hplot.Ticks{N: 15}

	// Open the desired files
	file, err := os.Open(LogPath + "ping_report.txt")
	errMgmt(err)

	var values plotter.XYs
	var firstTs float64
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "time=") && strings.Contains(line, " ms") {
			lineTs := line[1:strings.Index(line, "]")]
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
	sort.Slice(values, func(i, j int) bool {
		return values[i].Y < values[j].Y
	})
	toRemove := len(values) / 100
	values = values[:len(values)-toRemove]
	sort.Slice(values, func(i, j int) bool {
		return values[i].X < values[j].X
	})
	err = plotutil.AddLines(p, "Ping RTT", values)

	if err := p.Save(1500, 1000, LogPath+"pingPlot.pdf"); err != nil {
		panic(err)
	}
}

func TCPdumpPlotter(runs int) {
	for i := 1; i <= runs; i++ {
		fmt.Println("Plotting TCP RTT run: " + strconv.Itoa(i))
		p, err := plot.New()
		errMgmt(err)

		p.X.Label.Text = "Time (s)"
		p.Y.Label.Text = "TCP RTT (ms)"
		p.Title.Text = "TCP ACK Latency"
		p.Y.Tick.Marker = hplot.Ticks{N: 15}
		p.X.Tick.Marker = hplot.Ticks{N: 15}

		fileOtp, err := exec.Command("tshark",
			"-r", LogPath+strconv.Itoa(i)+"-tcpdump_report.pcap",
			"-Y", "tcp.analysis.ack_rtt and ip.dst==172.0.0.0/8",
			"-e", "frame.time_epoch",
			"-e", "tcp.analysis.ack_rtt",
			"-T", "fields",
			"-E", "separator=,",
			"-E", "quote=d").Output()
		errMgmt(err)

		var values plotter.XYs
		var firstTs float64
		records, _ := csv.NewReader(bytes.NewReader(fileOtp)).ReadAll()
		for _, row := range records {
			ts, fail := strconv.ParseFloat(row[0], 64)
			if fail != nil {
				continue
			}
			rtt, fail := strconv.ParseFloat(row[1], 64)
			if fail != nil {
				continue
			}
			if len(values) == 0 {
				firstTs = ts
			}
			values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
		}

		err = plotutil.AddLines(p, "ACK RTT", values)

		if err := p.Save(1500, 1000, LogPath+strconv.Itoa(i)+"-tcpPlot.pdf"); err != nil {
			panic(err)
		}
	}
}
