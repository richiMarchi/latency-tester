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
	"sync"
)

func PingPlotter(settings Settings, wg *sync.WaitGroup) {
	fmt.Println("Plotting Ping")

	pdfToSave := vgpdf.New(vg.Points(2000), vg.Points(1000))
	w, err := os.Create(settings.ExecDir + PlotDirName + "pingPlot.pdf")
	if err != nil {
		panic(err)
	}

	for i, dest := range settings.PingDestinations {
		if i != 0 {
			pdfToSave.NextPage()
		}

		// Open the desired file
		file, err := os.Open(settings.ExecDir + "ping_" + dest.Name + ".txt")
		errMgmt(err)

		p, err := plot.New()
		errMgmt(err)
		p.X.Label.Text = "Time (s)"
		p.Y.Label.Text = "OS RTT (ms)"
		p.Title.Text = "Ping destination: " + dest.Name
		p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
		p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}

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
		// Remove the last three percentiles
		sort.Slice(values, func(i, j int) bool {
			return values[i].Y < values[j].Y
		})
		toRemove := len(values) / 100
		values = values[:len(values)-toRemove*settings.PercentilesToRemove]
		sort.Slice(values, func(i, j int) bool {
			return values[i].X < values[j].X
		})
		err = plotutil.AddLines(p, "Ping RTT", values)
		p.Draw(draw.New(pdfToSave))
	}

	if _, err := pdfToSave.WriteTo(w); err != nil {
		panic(err)
	}
	w.Close()

	wg.Done()
}

func TcpdumpPlotter(settings Settings, run int, wg *sync.WaitGroup) {
	fmt.Println("Plotting TCP run #", run)

	// Open the desired file
	file, err := os.Open(settings.ExecDir + strconv.Itoa(run) + "-tcpdump_report.csv")
	errMgmt(err)

	var values plotter.XYs
	var firstTs float64
	var previousStream int
	streamCounter := 0
	// Read the file as CSV and remove the headers line
	records, _ := csv.NewReader(file).ReadAll()
	records = records[1:]

	pdfToSave := vgpdf.New(vg.Points(2000), vg.Points(1000))
	w, err := os.Create(settings.ExecDir + PlotDirName + strconv.Itoa(run) + "-tcpPlot.pdf")
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
		streamId, _ := strconv.Atoi(row[2])
		if len(values) == 0 {
			firstTs = ts
			previousStream = streamId
		}
		if previousStream != streamId || index == len(records)-1 {
			// If it is the last iteration, add the last record before saving to pdf
			if index == len(records)-1 {
				// Convert values to ms
				values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
			}
			p, err := plot.New()
			errMgmt(err)
			p.X.Label.Text = "Time (s)"
			p.Y.Label.Text = "TCP RTT (ms)"
			p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
			p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
			p.Title.Text = getTcpPlotTitle(settings, streamCounter)
			// Remove the last 3 percentiles
			sort.Slice(values, func(i, j int) bool {
				return values[i].Y < values[j].Y
			})
			toRemove := len(values) / 100
			values = values[:len(values)-toRemove*settings.PercentilesToRemove]
			sort.Slice(values, func(i, j int) bool {
				return values[i].X < values[j].X
			})
			err = plotutil.AddLines(p, "ACK RTT", values)
			if !(p.X.Max-p.X.Min < (float64(settings.RunsStepDuration) - (float64(settings.RunsStepDuration) / 10))) {
				if streamCounter != 0 {
					pdfToSave.NextPage()
				}
				p.Draw(draw.New(pdfToSave))
				streamCounter += 1
			}
			values = values[:0]
			previousStream = streamId
		}
		// Convert values to ms
		values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
	}

	if _, err := pdfToSave.WriteTo(w); err != nil {
		panic(err)
	}
	w.Close()

	wg.Done()
}

func RttPlotter(settings Settings, wg *sync.WaitGroup) {
	fmt.Println("Plotting E2E RTT")
	pdfToSave := vgpdf.New(vg.Points(2000), vg.Points(1000))
	w, err := os.Create(settings.ExecDir + PlotDirName + "e2eLatency.pdf")
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
								// Convert values to ms
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
				// Remove the last three percentiles
				sort.Slice(values, func(i, j int) bool {
					return values[i].Y < values[j].Y
				})
				toRemove := len(values) / 100
				values = values[:len(values)-toRemove*settings.PercentilesToRemove]
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

	wg.Done()
}
