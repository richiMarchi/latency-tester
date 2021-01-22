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
	"time"
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
	hourlyPdfToSave := vgpdf.New(vg.Points(2000), vg.Points(1000))
	hourly, err := os.Create(settings.ExecDir + PlotDirName + "e2eLatencyHourlyBoxplot.pdf")
	if err != nil {
		panic(err)
	}

	requestedRuns := requestedSlice(settings)
	for epIndex, addr := range settings.Endpoints {
		for interIndex, inter := range settings.Intervals {
			for sizeIndex, size := range settings.MsgSizes {

				var values plotter.XYs
				var runInterruptions []*hplot.VertLine
				hourlyMap := make(map[string]plotter.Values)
				var absoluteFirst float64
				var lastOfRun float64
				var runTime string
				for runIndex, run := range requestedRuns {
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
									intTs, _ := strconv.ParseInt(row[0], 10, 64)
									utcTs := time.Unix(0, intTs)
									runTime = fmt.Sprintf("%02d", run) + ") " + strconv.Itoa(utcTs.Hour()) + ":" +
										strconv.Itoa(utcTs.Minute())
								}
								// Convert values to ms
								xValue := (timeInter - absoluteFirst - runGap) / 1000000000
								values = append(values, plotter.XY{X: xValue, Y: parsed})
								hourlyMap[runTime] = append(hourlyMap[runTime], parsed)
								if i == len(records)-1 {
									lastOfRun = timeInter - runGap
									// Save X of last record of the run, to divide all with a vertical line in the plot
									if (runIndex + 1) != len(requestedRuns) {
										runInterruptions = append(runInterruptions, hplot.VLine(xValue, nil, nil))
									}
								}
							}
						}
					}
					if (runIndex+1)%12 == 0 || (runIndex+1) == len(requestedRuns) {
						if (epIndex+interIndex+sizeIndex) != 0 || (runIndex+1) > 12 {
							hourlyPdfToSave.NextPage()
						}
						box, err := plot.New()
						errMgmt(err)
						box.X.Label.Text = "UTC Time (hh:mm)"
						box.Y.Label.Text = "E2E RTT (ms)"
						box.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
						box.Title.Text = "E2E Latency: " + addr.Description + " - " + strconv.Itoa(inter) + "ms - " + strconv.Itoa(size) + "B"
						boxplot, _, _ := generateStringBoxPlotAndLimits(box, &hourlyMap, settings.PercentilesToRemove)
						if settings.RttMin != 0 {
							boxplot.Y.Min = settings.RttMin
						}
						if settings.RttMax != 0 {
							boxplot.Y.Max = settings.RttMax
						}
						boxplot.Draw(draw.New(hourlyPdfToSave))
						hourlyMap = make(map[string]plotter.Values)
					}
				}
				if (epIndex + interIndex + sizeIndex) != 0 {
					pdfToSave.NextPage()
				}
				// Standard Plot
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
				for _, line := range runInterruptions {
					p.Add(line)
				}
				if settings.RttMin != 0 {
					p.Y.Min = settings.RttMin
				}
				if settings.RttMax != 0 {
					p.Y.Max = settings.RttMax
				}
				p.Draw(draw.New(pdfToSave))
			}
		}
	}
	if _, err := pdfToSave.WriteTo(w); err != nil {
		panic(err)
	}
	w.Close()
	if _, err := hourlyPdfToSave.WriteTo(hourly); err != nil {
		panic(err)
	}
	hourly.Close()

	wg.Done()
}
