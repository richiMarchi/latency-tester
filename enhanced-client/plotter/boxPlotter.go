package main

import (
	"encoding/csv"
	"fmt"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"math"
	"sort"
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
					settings.ExecDir, settings.PercentilesToRemove)
				filename = "endpointsBoxPlot"
			case INTERVALS:
				plots[i][j], tmpMin, tmpMax = sizeXepBoxPlot(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals,
					settings.ExecDir, settings.PercentilesToRemove)
				filename = "intervalsBoxPlot"
			case SIZES:
				plots[i][j], tmpMin, tmpMax = intXepBoxPlot(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes,
					settings.ExecDir, settings.PercentilesToRemove)
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
		standardMin = min - 1
	}
	if settings.RttMax != 0 {
		standardMax = settings.RttMax
	} else {
		standardMax = max + 3
	}
	adjustMinMaxY(plots, rows, cols, standardMin, standardMax)
	commonPlotting(plots, rows, cols, 100+cols*elems*200, settings.ExecDir+PlotDirName+filename)

	wg.Done()
}

// Return a boxplot of the e2e rtt of the sizes given the interval and the endpoint
func intXepBoxPlot(ep EndpointData, si int, msgSizes []int, execdir string, percentilesToRemove int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for " + ep.Description + " and send interval " + strconv.Itoa(si))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, "-"+ep.Destination+".i"+strconv.Itoa(si)+".x")

	valuesMap := make(map[int]plotter.Values)

	for _, f := range openFiles {
		parsedSizeVal, err := strconv.ParseInt(f.Name()[strings.LastIndex(f.Name(), "x")+1:len(f.Name())-4], 10, 32)
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

	p.X.Label.Text = "Request Size (KiB)"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"

	return generateBoxPlotAndLimits(p, &valuesMap, percentilesToRemove)
}

// Return a boxplot of the e2e rtt of the intervals given the size and the endpoint
func sizeXepBoxPlot(ep EndpointData, msgSize int, sis []int, execdir string, percentilesToRemove int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for message size " + strconv.Itoa(msgSize) + " and endpoint " + ep.Description)
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, "-"+ep.Destination+".i", ".x"+strconv.Itoa(msgSize)+".csv")

	valuesMap := make(map[int]plotter.Values)

	for _, f := range openFiles {
		parsedInterVal, err := strconv.ParseInt(f.Name()[strings.LastIndex(f.Name(), ".i")+2:strings.LastIndex(f.Name(), ".x")], 10, 32)
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

	p.X.Label.Text = "Send Interval (ms)"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "KiB"

	return generateBoxPlotAndLimits(p, &valuesMap, percentilesToRemove)
}

// Return a boxplot of the e2e rtt of the endpoints given the interval and the size
func intXsizeBoxPlot(msgSize int,
	si int,
	eps []EndpointData,
	execdir string,
	percentilesToRemove int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for interval " + strconv.Itoa(si) + " and message size " + strconv.Itoa(msgSize))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, ".i"+strconv.Itoa(si)+".x"+strconv.Itoa(msgSize)+".csv")

	valuesMap := make(map[string]plotter.Values)

	for _, f := range openFiles {
		parsedSizeVal := f.Name()[strings.Index(f.Name(), "-")+1 : strings.LastIndex(f.Name(), ".i")]
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

	p.X.Label.Text = "Endpoint"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = strconv.Itoa(si) + "ms - " + strconv.Itoa(msgSize) + "KiB"

	// Get map ordered keys
	keys := make([]string, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for _, k := range keys {
		// Remove the first three and last three percentiles in order to avoid unreadable plots
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][toRemove*percentilesToRemove : len(valuesMap[k])-toRemove*percentilesToRemove]
		boxplot, err := plotter.NewBoxPlot(w, position, valuesMap[k])
		errMgmt(err)
		nominals = append(nominals, k+" (Median:"+strconv.FormatFloat(boxplot.Median, 'f', 2, 64)+")")
		mins = append(mins, boxplot.AdjLow)
		maxes = append(maxes, boxplot.AdjHigh)
		position += 1
		p.Add(boxplot)
	}
	p.NominalX(nominals...)

	return p, floats.Min(mins), floats.Max(maxes)
}
