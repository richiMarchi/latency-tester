package main

import (
	"encoding/csv"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
)

func typedCDFs(settings Settings, objectType int, wg *sync.WaitGroup) {
	rows, cols, _ := getLoopElems(settings, objectType)
	min := math.Inf(1)
	max := math.Inf(-1)
	var filename string
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			switch objectType {
			case ENDPOINTS:
				plots[i][j] = intXsizeCDF(settings.MsgSizes[i], settings.Intervals[j], settings.Endpoints, settings.ExecDir,
					settings.PercentilesToRemove, requestedSlice(settings))
				filename = "endpointsCDF"
			case INTERVALS:
				plots[i][j] = sizeXepCDF(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals, settings.ExecDir,
					settings.PercentilesToRemove, requestedSlice(settings))
				filename = "intervalsCDF"
			case SIZES:
				plots[i][j] = intXepCDF(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes, settings.ExecDir,
					settings.PercentilesToRemove, requestedSlice(settings))
				filename = "sizesCDF"
			default:
				panic("Wrong objectType in loop elements: only values 0,1 and 2 are allowed")
			}
			min = floats.Min([]float64{min, plots[i][j].X.Min})
			max = floats.Max([]float64{max, plots[i][j].X.Max})
		}
	}

	if settings.RttMin != 0 {
		min = settings.RttMin
	}
	if settings.RttMax != 0 {
		max = settings.RttMax
	}

	if !settings.EqualizationDisabled {
		adjustMinMaxX(plots, rows, cols, min, max)
	}
	commonPlotting(plots, rows, cols, cols*500, settings.ExecDir+PlotDirName+filename)

	wg.Done()
}

// Return a cdf of the e2e rtt of the sizes given the interval and the endpoint
func intXepCDF(
	ep EndpointData,
	si int,
	sizes []int,
	execdir string,
	percentilesToRemove int,
	requestedRuns []int) *plot.Plot {
	log.Println(LoggerHdr + "CDF for " + ep.Description + " and send interval " + strconv.Itoa(si))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, requestedRuns,
		"-"+strings.ReplaceAll(ep.Destination, ":", "_")+".i"+strconv.Itoa(si)+".x")

	valuesMap := make(map[int]plotter.Values)

	for _, f := range openFiles {
		filename := filenameOnly(f.Name())
		parsedSizeVal, err := strconv.ParseInt(filename[strings.LastIndex(filename, "x")+1:len(filename)-4], 10, 32)
		sizeVal := int(parsedSizeVal)
		errMgmt(err)
		if intInSlice(sizeVal, sizes) {
			records, _ := csv.NewReader(f).ReadAll()
			for i, row := range records {
				if i != 0 {
					parsed, fail := strconv.ParseFloat(row[2], 64)
					if fail != nil {
						continue
					}
					valuesMap[sizeVal] = append(valuesMap[sizeVal], parsed)
				}
			}
		}
	}

	p.X.Label.Text = "E2E RTT (ms)"
	p.Y.Label.Text = "P(x)"
	p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"
	p.Title.Font.Size = 20

	generateIntCDFPlot(p, &valuesMap, percentilesToRemove)

	return p
}

// Return a cdf of the e2e rtt of the intervals given the size and the endpoint
func sizeXepCDF(
	ep EndpointData,
	msgSize int,
	sis []int,
	execdir string,
	percentilesToRemove int,
	requestedRuns []int) *plot.Plot {
	log.Println(LoggerHdr + "CDF for " + ep.Description + " and message size " + strconv.Itoa(msgSize))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, requestedRuns, "-"+strings.ReplaceAll(ep.Destination, ":", "_")+".i",
		".x"+strconv.Itoa(msgSize)+".csv")

	valuesMap := make(map[int]plotter.Values)

	for _, f := range openFiles {
		filename := filenameOnly(f.Name())
		parsedInterVal, err := strconv.ParseInt(
			filename[strings.LastIndex(filename, ".i")+2:strings.LastIndex(filename, ".x")], 10, 32)
		interVal := int(parsedInterVal)
		errMgmt(err)
		if intInSlice(interVal, sis) {
			records, _ := csv.NewReader(f).ReadAll()
			for i, row := range records {
				if i != 0 {
					parsed, fail := strconv.ParseFloat(row[2], 64)
					if fail != nil {
						continue
					}
					valuesMap[interVal] = append(valuesMap[interVal], parsed)
				}
			}
		}
	}

	p.X.Label.Text = "E2E RTT (ms)"
	p.Y.Label.Text = "P(x)"
	p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "B"
	p.Title.Font.Size = 20

	generateIntCDFPlot(p, &valuesMap, percentilesToRemove)

	return p
}

// Return a cdf of the e2e rtt of the endpoints given the interval and the size
func intXsizeCDF(
	msgSize int,
	si int, eps []EndpointData,
	execdir string,
	percentilesToRemove int,
	requestedRuns []int) *plot.Plot {
	log.Println(LoggerHdr + "CDF for interval " + strconv.Itoa(si) + " and message size " + strconv.Itoa(msgSize))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, requestedRuns, ".i"+strconv.Itoa(si)+".x"+strconv.Itoa(msgSize)+".csv")

	valuesMap := make(map[string]plotter.Values)

	for _, f := range openFiles {
		filename := filenameOnly(f.Name())
		parsedSizeVal := filename[strings.Index(filename, "-")+1 : strings.LastIndex(filename, ".i")]
		description, present := nameFromDest(parsedSizeVal, &eps)
		if present {
			records, _ := csv.NewReader(f).ReadAll()
			for i, row := range records {
				if i != 0 {
					parsed, fail := strconv.ParseFloat(row[2], 64)
					if fail != nil {
						continue
					}
					valuesMap[description] = append(valuesMap[description], parsed)
				}
			}
		}
	}

	p.X.Label.Text = "E2E RTT (ms)"
	p.Y.Label.Text = "P(x)"
	p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = strconv.Itoa(si) + "ms - " + strconv.Itoa(msgSize) + "B"
	p.Title.Font.Size = 20

	generateStringCDFPlot(p, &valuesMap, percentilesToRemove)

	return p
}
