package main

import (
	"encoding/csv"
	"fmt"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

func SizesCDF(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.Intervals)
	min := math.Inf(1)
	max := math.Inf(-1)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			plots[i][j] = intXepCDF(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes)
			min = floats.Min([]float64{min, plots[i][j].X.Min})
			max = floats.Max([]float64{max, plots[i][j].X.Max})
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].X.Min = min
			plots[i][j].X.Max = max
		}
	}

	commonPlotting(plots, rows, cols, cols*500, "sizesCDF")
}

func intXepCDF(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, si int, sizes []int) *plot.Plot {
	fmt.Println("CDF for " + ep.Description + " and send interval " + strconv.Itoa(si))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	files, err := ioutil.ReadDir(LogPath)
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), "-"+ep.Destination+".i"+strconv.Itoa(si)+".x") {
			file, err := os.Open(LogPath + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			openFiles = append(openFiles, file)
		}
	}

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
	p.X.Tick.Marker = hplot.Ticks{N: 15}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"

	// Get map ordered keys
	keys := make([]int, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var lines []interface{}
	for _, k := range keys {
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][:len(valuesMap[k])-toRemove*2]
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len(valuesMap[k])) {
			toAdd = append(toAdd, plotter.XY{X: valuesMap[k][i], Y: y})
		}
		lines = append(lines, strconv.Itoa(k))
		lines = append(lines, toAdd)
	}
	err = plotutil.AddLines(p, lines...)
	errMgmt(err)

	return p
}

func IntervalsCDF(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.MsgSizes)
	min := math.Inf(1)
	max := math.Inf(-1)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			plots[i][j] = sizeXepCDF(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals)
			min = floats.Min([]float64{min, plots[i][j].X.Min})
			max = floats.Max([]float64{max, plots[i][j].X.Max})
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].X.Min = min
			plots[i][j].X.Max = max
		}
	}

	commonPlotting(plots, rows, cols, cols*500, "intervalsCDF")
}

func sizeXepCDF(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, msgSize int, sis []int) *plot.Plot {
	fmt.Println("CDF for " + ep.Description + " and message size " + strconv.Itoa(msgSize))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	files, err := ioutil.ReadDir(LogPath)
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), "-"+ep.Destination+".i") && strings.Contains(f.Name(), ".x"+strconv.Itoa(msgSize)+".csv") {
			file, err := os.Open(LogPath + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			openFiles = append(openFiles, file)
		}
	}

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
	p.X.Tick.Marker = hplot.Ticks{N: 15}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "KiB"

	// Get map ordered keys
	keys := make([]int, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var lines []interface{}
	for _, k := range keys {
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][:len(valuesMap[k])-toRemove*2]
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len(valuesMap[k])) {
			toAdd = append(toAdd, plotter.XY{X: valuesMap[k][i], Y: y})
		}
		lines = append(lines, strconv.Itoa(k))
		lines = append(lines, toAdd)
	}
	err = plotutil.AddLines(p, lines...)
	errMgmt(err)

	return p
}

func EndpointsCDF(settings Settings) {
	rows := len(settings.MsgSizes)
	cols := len(settings.Intervals)
	min := math.Inf(1)
	max := math.Inf(-1)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			plots[i][j] = intXsizeCDF(settings.MsgSizes[i], settings.Intervals[j], settings.Endpoints)
			min = floats.Min([]float64{min, plots[i][j].X.Min})
			max = floats.Max([]float64{max, plots[i][j].X.Max})
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].X.Min = min
			plots[i][j].X.Max = max
		}
	}

	commonPlotting(plots, rows, cols, cols*500, "endpointsCDF")
}

func intXsizeCDF(msgSize int, si int, eps []struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}) *plot.Plot {
	fmt.Println("CDF for interval " + strconv.Itoa(si) + " and message size " + strconv.Itoa(msgSize))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	files, err := ioutil.ReadDir(LogPath)
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), ".i"+strconv.Itoa(si)+".x"+strconv.Itoa(msgSize)+".csv") {
			file, err := os.Open(LogPath + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			openFiles = append(openFiles, file)
		}
	}

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
	p.X.Tick.Marker = hplot.Ticks{N: 15}
	p.Title.Text = strconv.Itoa(si) + "ms - " + strconv.Itoa(msgSize) + "KiB"

	// Get map ordered keys
	keys := make([]string, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []interface{}
	for _, k := range keys {
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][:len(valuesMap[k])-toRemove*2]
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
