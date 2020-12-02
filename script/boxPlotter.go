package main

import (
	"encoding/csv"
	"fmt"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

func SizesBoxPlot(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.Intervals)
	elems := len(settings.MsgSizes)
	min := math.Inf(1)
	max := math.Inf(-1)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			var tmpMin float64
			var tmpMax float64
			plots[i][j], tmpMin, tmpMax = intXepBoxPlot(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes)
			min = floats.Min([]float64{min, tmpMin})
			max = floats.Max([]float64{max, tmpMax})
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].Y.Min = min - 1
			plots[i][j].Y.Max = max + 3
		}
	}

	commonPlotting(plots, rows, cols, cols*elems*200, "sizesBoxPlot")
}

func IntervalsBoxPlot(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.MsgSizes)
	elems := len(settings.Intervals)
	min := math.Inf(1)
	max := math.Inf(-1)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			var tmpMin float64
			var tmpMax float64
			plots[i][j], tmpMin, tmpMax = sizeXepBoxPlot(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals)
			min = floats.Min([]float64{min, tmpMin})
			max = floats.Max([]float64{max, tmpMax})
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].Y.Min = min - 1
			plots[i][j].Y.Max = max + 3
		}
	}

	commonPlotting(plots, rows, cols, cols*elems*200, "intervalsBoxPlot")
}

func EndpointsBoxPlot(settings Settings) {
	rows := len(settings.MsgSizes)
	cols := len(settings.Intervals)
	elems := len(settings.Endpoints)
	min := math.Inf(1)
	max := math.Inf(-1)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			var tmpMin float64
			var tmpMax float64
			plots[i][j], tmpMin, tmpMax = intXsizeBoxPlot(settings.MsgSizes[i], settings.Intervals[j], settings.Endpoints)
			min = floats.Min([]float64{min, tmpMin})
			max = floats.Max([]float64{max, tmpMax})
		}
	}

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].Y.Min = min - 1
			plots[i][j].Y.Max = max + 3
		}
	}

	commonPlotting(plots, rows, cols, cols*elems*200, "endpointsBoxPlot")
}

func intXepBoxPlot(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, si int, msgSizes []int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for " + ep.Description + " and send interval " + strconv.Itoa(si))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	files, err := ioutil.ReadDir("/tmp")
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), "-"+ep.Destination+".i"+strconv.Itoa(si)+".x") {
			file, err := os.Open("/tmp/" + f.Name())
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
	p.Y.Tick.Marker = commaTicks{}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"

	// Get map ordered keys
	keys := make([]int, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for _, k := range keys {
		boxplot, err := plotter.NewBoxPlot(w, position, valuesMap[k])
		errMgmt(err)
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][toRemove*3 : len(valuesMap[k])-toRemove*3]
		nominals = append(nominals, strconv.Itoa(k)+" (Median:"+strconv.FormatFloat(boxplot.Median, 'f', 2, 64)+")")
		mins = append(mins, boxplot.AdjLow)
		maxes = append(maxes, boxplot.AdjHigh)
		position += 1
		p.Add(boxplot)
	}
	p.NominalX(nominals...)

	return p, floats.Min(mins), floats.Max(maxes)
}

func sizeXepBoxPlot(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, msgSize int, sis []int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for message size " + strconv.Itoa(msgSize) + " and endpoint " + ep.Description)
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	files, err := ioutil.ReadDir("/tmp")
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), "-"+ep.Destination+".i") && strings.Contains(f.Name(), ".x"+strconv.Itoa(msgSize)+".csv") {
			file, err := os.Open("/tmp/" + f.Name())
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

	p.X.Label.Text = "Send Interval (ms)"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = commaTicks{}
	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "KiB"

	// Get map ordered keys
	keys := make([]int, 0, len(valuesMap))
	for k := range valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for _, k := range keys {
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][toRemove*3 : len(valuesMap[k])-toRemove*3]
		boxplot, err := plotter.NewBoxPlot(w, position, valuesMap[k])
		errMgmt(err)
		nominals = append(nominals, strconv.Itoa(k)+" (Median:"+strconv.FormatFloat(boxplot.Median, 'f', 2, 64)+")")
		mins = append(mins, boxplot.AdjLow)
		maxes = append(maxes, boxplot.AdjHigh)
		position += 1
		p.Add(boxplot)
	}
	p.NominalX(nominals...)

	return p, floats.Min(mins), floats.Max(maxes)
}

func intXsizeBoxPlot(msgSize int, si int, eps []struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for interval " + strconv.Itoa(si) + " and message size " + strconv.Itoa(msgSize))
	p, err := plot.New()
	errMgmt(err)

	// Open the desired files
	files, err := ioutil.ReadDir("/tmp")
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), ".i"+strconv.Itoa(si)+".x"+strconv.Itoa(msgSize)+".csv") {
			file, err := os.Open("/tmp/" + f.Name())
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

	p.X.Label.Text = "Endpoint"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.Y.Tick.Marker = commaTicks{}
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
		sort.Float64s(valuesMap[k])
		toRemove := len(valuesMap[k]) / 100
		valuesMap[k] = valuesMap[k][toRemove*3 : len(valuesMap[k])-toRemove*3]
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
