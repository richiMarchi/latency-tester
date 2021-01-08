package main

import (
	"bufio"
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
	"sort"
	"strconv"
	"strings"
)

func PingPlotter(settings Settings) {
	fmt.Println("Plotting Ping")
	p, err := plot.New()
	errMgmt(err)

	p.X.Label.Text = "Time (s)"
	p.Y.Label.Text = "OS RTT (ms)"
	p.Title.Text = "Ping destination: " + settings.PingIp
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}

	// Open the desired file
	file, err := os.Open(settings.ExecDir + "ping_report.txt")
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
	values = values[:len(values)-toRemove*3]
	sort.Slice(values, func(i, j int) bool {
		return values[i].X < values[j].X
	})
	err = plotutil.AddLines(p, "Ping RTT", values)

	if err := p.Save(1500, 1000, settings.ExecDir+"pingPlot.pdf"); err != nil {
		panic(err)
	}
}

func RttPlotter(settings Settings) {
	fmt.Println("Plotting E2E RTT")
	pdfToSave := vgpdf.New(vg.Points(2000), vg.Points(1000))
	w, err := os.Create(settings.ExecDir + "e2eLatency.pdf")
	if err != nil {
		panic(err)
	}

	for epIndex, addr := range settings.Endpoints {
		for interIndex, inter := range settings.Intervals {
			for sizeIndex, size := range settings.MsgSizes {

				var values plotter.XYs
				var absoluteFirst float64
				var lastOfRun float64
				for run := 1; run <= settings.Runs; run++ {
					file, err := os.Open(settings.ExecDir +
						strconv.Itoa(run) + "-" + addr.Destination + ".i" + strconv.Itoa(inter) + ".x" + strconv.Itoa(size) + ".csv")
					if err == nil {
						records, _ := csv.NewReader(file).ReadAll()
						var runGap float64
						for i, row := range records {
							if i != 0 {
								parsed, fail := strconv.ParseFloat(row[2], 64)
								if fail != nil {
									continue
								}
								timeInter, fail := strconv.ParseFloat(row[0], 64)
								if fail != nil {
									continue
								}
								if i == 1 {
									if run == 1 {
										absoluteFirst = timeInter
										lastOfRun = timeInter
									}
									runGap = timeInter - lastOfRun
								}
								values = append(values, plotter.XY{X: (timeInter - absoluteFirst - runGap) / 1000000000, Y: parsed})
								if i == len(records)-1 {
									lastOfRun = timeInter - runGap
								}
							}
						}
					}
				}
				if (epIndex + interIndex + sizeIndex) != 0 {
					pdfToSave.NextPage()
				}
				p, err := plot.New()
				errMgmt(err)
				p.X.Label.Text = "Time (s)"
				p.Y.Label.Text = "E2E RTT (ms)"
				p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
				p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
				p.Title.Text = "E2E Latency: " + addr.Description + " - " + strconv.Itoa(inter) + "ms - " + strconv.Itoa(size) + "B"
				sort.Slice(values, func(i, j int) bool {
					return values[i].Y < values[j].Y
				})
				toRemove := len(values) / 100
				values = values[:len(values)-toRemove*3]
				sort.Slice(values, func(i, j int) bool {
					return values[i].X < values[j].X
				})
				err = plotutil.AddLines(p, "RTT", values)
				p.Draw(draw.New(pdfToSave))
			}
		}
	}
	if _, err := pdfToSave.WriteTo(w); err != nil {
		panic(err)
	}
	w.Close()
}
