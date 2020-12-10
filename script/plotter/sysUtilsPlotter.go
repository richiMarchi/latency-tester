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
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgpdf"
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

func TCPdumpPlotter(settings Settings) {
	for i := 1; i <= settings.Runs; i++ {
		fmt.Println("Plotting TCP RTT run: " + strconv.Itoa(i))

		fileOtp, err := exec.Command("tshark",
			"-r", LogPath+strconv.Itoa(i)+"-tcpdump_report.pcap",
			"-Y", "tcp.analysis.ack_rtt and ip.dst==172.0.0.0/8",
			"-e", "frame.time_epoch",
			"-e", "tcp.analysis.ack_rtt",
			"-e", "tcp.stream",
			"-T", "fields",
			"-E", "separator=,",
			"-E", "quote=d").Output()
		errMgmt(err)

		var values plotter.XYs
		var firstTs float64
		var previousStream = 0
		records, _ := csv.NewReader(bytes.NewReader(fileOtp)).ReadAll()

		pdfToSave := vgpdf.New(vg.Points(1500), vg.Points(1000))
		w, err := os.Create(LogPath + strconv.Itoa(i) + "-tcpPlot.pdf")
		if err != nil {
			panic(err)
		}

		for index, row := range records {
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
			streamId, _ := strconv.Atoi(row[2])
			if previousStream != streamId || index == len(records)-1 {
				if previousStream != 0 {
					pdfToSave.NextPage()
				}
				// If it is the last iteration, add the last record before saving to pdf
				if index == len(records)-1 {
					values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
				}
				p, err := plot.New()
				errMgmt(err)
				p.X.Label.Text = "Time (s)"
				p.Y.Label.Text = "TCP RTT (ms)"
				p.Y.Tick.Marker = hplot.Ticks{N: 15}
				p.X.Tick.Marker = hplot.Ticks{N: 15}
				for y, addr := range settings.Endpoints {
					for j, inter := range settings.Intervals {
						for k, size := range settings.MsgSizes {
							if y+j+k == previousStream {
								p.Title.Text = "TCP ACK Latency: " + addr.Description + " - " + strconv.Itoa(inter) + "ms - " + strconv.Itoa(size) + "B"
							}
						}
					}
				}
				err = plotutil.AddLines(p, "ACK RTT", values)
				p.Draw(draw.New(pdfToSave))
				values = values[:0]
				previousStream = streamId
			}
			values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
		}

		if _, err := pdfToSave.WriteTo(w); err != nil {
			panic(err)
		}

		w.Close()

		/*if err := p.Save(1500, 1000, LogPath+strconv.Itoa(i)+"-tcpPlot.pdf"); err != nil {
			panic(err)
		}*/
	}
}
