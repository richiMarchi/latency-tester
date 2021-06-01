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

func typedBoxPlots(settings Settings, objectType int, wg *sync.WaitGroup) {
	rows, cols, elems := getLoopElems(settings, objectType)
	min := math.Inf(1)
	max := math.Inf(-1)
	var filename string
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			var tmpMin, tmpMax float64
			switch objectType {
			case ENDPOINTS:
				plots[i][j], tmpMin, tmpMax = intXsizeBoxPlot(settings.MsgSizes[i], settings.Intervals[j], settings.Endpoints,
					settings.ExecDir, settings.PercentilesToRemove, settings.WhiskerMin, settings.WhiskerMax,
					requestedSlice(settings))
				filename = "endpointsBoxPlot"
			case INTERVALS:
				plots[i][j], tmpMin, tmpMax = sizeXepBoxPlot(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals,
					settings.ExecDir, settings.PercentilesToRemove, settings.WhiskerMin, settings.WhiskerMax, requestedSlice(settings))
				filename = "intervalsBoxPlot"
			case SIZES:
				plots[i][j], tmpMin, tmpMax = intXepBoxPlot(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes,
					settings.ExecDir, settings.PercentilesToRemove, settings.WhiskerMin, settings.WhiskerMax,
					requestedSlice(settings))
				filename = "sizesBoxPlot"
			default:
				panic("Wrong objectType in loop elements: only values 0,1 and 2 are allowed")
			}
			min = floats.Min([]float64{min, tmpMin})
			max = floats.Max([]float64{max, tmpMax})
		}
	}

	var standardMin float64
	var standardMax float64
	if settings.RttMin != 0 {
		standardMin = settings.RttMin
	} else {
		standardMin = min
	}
	if settings.RttMax != 0 {
		standardMax = settings.RttMax
	} else {
		standardMax = max
	}
	if !settings.EqualizationDisabled {
		adjustMinMaxY(plots, rows, cols, standardMin, standardMax)
	}
	commonPlotting(plots, rows, cols, 100+cols*elems*200, settings.ExecDir+PlotDirName+filename)

	wg.Done()
}

// Return a boxplot of the e2e rtt of the sizes given the interval and the endpoint
func intXepBoxPlot(ep EndpointData,
	si int,
	msgSizes []int,
	execdir string,
	percentilesToRemove int,
	whiskerMin int,
	whiskerMax int,
	requestedRuns []int) (*plot.Plot, float64, float64) {
	log.Println(LoggerHdr + "Plot for " + ep.Description + " and send interval " + strconv.Itoa(si))
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
		if intInSlice(sizeVal, msgSizes) {
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

	closeOpenFiles(openFiles)

	p.X.Label.Text = "Request Size (B)"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"
	configurePlotFontSizesMultiple(p, true)

	return generateIntBoxPlotAndLimits(p, &valuesMap, percentilesToRemove, whiskerMin, whiskerMax)
}

// Return a boxplot of the e2e rtt of the intervals given the size and the endpoint
func sizeXepBoxPlot(ep EndpointData,
	msgSize int,
	sis []int,
	execdir string,
	percentilesToRemove int,
	whiskerMin int,
	whiskerMax int,
	requestedRuns []int) (*plot.Plot, float64, float64) {
	log.Println(LoggerHdr + "Plot for message size " + strconv.Itoa(msgSize) + " and endpoint " + ep.Description)
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

	closeOpenFiles(openFiles)

	p.X.Label.Text = "Send Interval (ms)"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "B"
	configurePlotFontSizesMultiple(p, true)

	return generateIntBoxPlotAndLimits(p, &valuesMap, percentilesToRemove, whiskerMin, whiskerMax)
}

// Return a boxplot of the e2e rtt of the endpoints given the interval and the size
func intXsizeBoxPlot(msgSize int,
	si int,
	eps []EndpointData,
	execdir string,
	percentilesToRemove int,
	whiskerMin int,
	whiskerMax int,
	requestedRuns []int) (*plot.Plot, float64, float64) {
	log.Println(LoggerHdr + "Plot for interval " + strconv.Itoa(si) + " and message size " + strconv.Itoa(msgSize))
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

	closeOpenFiles(openFiles)

	p.X.Label.Text = "Endpoint"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = strconv.Itoa(si) + "ms - " + strconv.Itoa(msgSize) + "B"
	configurePlotFontSizesMultiple(p, true)

	return generateStringBoxPlotAndLimits(p, &valuesMap, percentilesToRemove, whiskerMin, whiskerMax)
}
