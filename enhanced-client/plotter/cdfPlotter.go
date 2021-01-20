package main

import (
	"encoding/csv"
	"fmt"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"math"
	"sort"
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
					settings.PercentilesToRemove)
				filename = "endpointsCDF"
			case INTERVALS:
				plots[i][j] = sizeXepCDF(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals, settings.ExecDir,
					settings.PercentilesToRemove)
				filename = "intervalsCDF"
			case SIZES:
				plots[i][j] = intXepCDF(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes, settings.ExecDir,
					settings.PercentilesToRemove)
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
	adjustMinMaxX(plots, rows, cols, min, max)
	commonPlotting(plots, rows, cols, cols*500, settings.ExecDir+PlotDirName+filename)

	wg.Done()
}

// Return a cdf of the e2e rtt of the sizes given the interval and the endpoint
func intXepCDF(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, si int, sizes []int, execdir string, percentilesToRemove int) *plot.Plot {
	fmt.Println("CDF for " + ep.Description + " and send interval " + strconv.Itoa(si))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	openFiles := openDesiredFiles(execdir, "-"+ep.Destination+".i"+strconv.Itoa(si)+".x")

	valuesMap := make(map[int]plotter.Values)

	for _, f := range openFiles {
		parsedSizeVal, err := strconv.ParseInt(f.Name()[strings.LastIndex(f.Name(), "x")+1:len(f.Name())-4], 10, 32)
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

	generateCDFPlot(p, &valuesMap, percentilesToRemove)

	return p
}

// Return a cdf of the e2e rtt of the intervals given the size and the endpoint
func sizeXepCDF(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, msgSize int, sis []int, execdir string, percentilesToRemove int) *plot.Plot {
	fmt.Println("CDF for " + ep.Description + " and message size " + strconv.Itoa(msgSize))
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

	p.X.Label.Text = "E2E RTT (ms)"
	p.Y.Label.Text = "P(x)"
	p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "KiB"

	generateCDFPlot(p, &valuesMap, percentilesToRemove)

	return p
}

// Return a cdf of the e2e rtt of the endpoints given the interval and the size
func intXsizeCDF(msgSize int, si int, eps []struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, execdir string, percentilesToRemove int) *plot.Plot {
	fmt.Println("CDF for interval " + strconv.Itoa(si) + " and message size " + strconv.Itoa(msgSize))
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

	p.X.Label.Text = "E2E RTT (ms)"
	p.Y.Label.Text = "P(x)"
	p.X.Tick.Marker = hplot.Ticks{N: AxisTicks}
	p.Title.Text = strconv.Itoa(si) + "ms - " + strconv.Itoa(msgSize) + "KiB"

	// Get map ordered keys
	keys := make([]string, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []interface{}
	for _, k := range keys {
		// Remove the last two percentiles in order to avoid unreadable plots
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][:len(valuesMap[k])-toRemove*percentilesToRemove]
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len(valuesMap[k])) {
			toAdd = append(toAdd, plotter.XY{X: valuesMap[k][i], Y: y})
		}
		lines = append(lines, k)
		lines = append(lines, toAdd)
	}
	err = plotutil.AddLines(p, lines...)
	errMgmt(err)

	return p
}
