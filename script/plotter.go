package main

import (
	"encoding/csv"
	"fmt"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

func SizesBoxPlot(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.Intervals)
	elems := len(settings.MsgSizes)
	var min float64 = 10000
	var max float64 = 0
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
			plots[i][j].Y.Max = max + 5
		}
	}

	commonPlotting(plots, rows, cols, elems*150, "sizesBoxPlot")
}

func IntervalsBoxPlot(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.MsgSizes)
	elems := len(settings.Intervals)
	var min float64 = 10000
	var max float64 = 0
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
			plots[i][j].Y.Max = max + 5
		}
	}

	commonPlotting(plots, rows, cols, elems*150, "intervalsBoxPlot")
}

func EndpointsBoxPlot(settings Settings) {
	rows := len(settings.MsgSizes)
	cols := len(settings.Intervals)
	elems := len(settings.Endpoints)
	var min float64 = 10000
	var max float64 = 0
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
			plots[i][j].Y.Max = max + 5
		}
	}

	commonPlotting(plots, rows, cols, elems*150, "endpointsBoxPlot")
}

func SizesCDF(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.Intervals)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			plots[i][j] = intXepCDF(settings.Endpoints[i], settings.Intervals[j], settings.MsgSizes)
		}
	}

	commonPlotting(plots, rows, cols, 500, "sizesCDF")
}

func intXepCDF(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, si int, sizes []int) *plot.Plot {
	fmt.Println("CDF for " + ep.Description + " and send interval " + strconv.Itoa(si))
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

	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"

	var lines []interface{}
	for k, elem := range valuesMap {
		sort.Float64s(elem)
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len(elem)) {
			toAdd = append(toAdd, plotter.XY{X: elem[i], Y: y})
		}
		lines = append(lines, strconv.Itoa(k))
		lines = append(lines, toAdd)
	}
	err = plotutil.AddLinePoints(p, lines...)
	errMgmt(err)

	return p
}

func IntervalsCDF(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.MsgSizes)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			plots[i][j] = sizeXepCDF(settings.Endpoints[i], settings.MsgSizes[j], settings.Intervals)
		}
	}

	commonPlotting(plots, rows, cols, 500, "intervalsCDF")
}

func sizeXepCDF(ep struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}, msgSize int, sis []int) *plot.Plot {
	fmt.Println("CDF for " + ep.Description + " and message size " + strconv.Itoa(msgSize))
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

	p.X.Label.Text = "E2E RTT (ms)"
	p.Y.Label.Text = "P(x)"

	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "KiB"

	var lines []interface{}
	for k, elem := range valuesMap {
		sort.Float64s(elem)
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len(elem)) {
			toAdd = append(toAdd, plotter.XY{X: elem[i], Y: y})
		}
		lines = append(lines, strconv.Itoa(k))
		lines = append(lines, toAdd)
	}
	err = plotutil.AddLinePoints(p, lines...)
	errMgmt(err)

	return p
}

func commonPlotting(plots [][]*plot.Plot, rows int, cols int, cardWidth int, filename string) {

	img := vgimg.New(vg.Points(float64(cardWidth*cols)), vg.Points(float64(rows*650)))
	dc := draw.New(img)

	t := draw.Tiles{
		Rows:      rows,
		Cols:      cols,
		PadX:      vg.Millimeter,
		PadY:      vg.Millimeter,
		PadTop:    vg.Points(2),
		PadBottom: vg.Points(2),
		PadLeft:   vg.Points(2),
		PadRight:  vg.Points(2),
	}

	canvases := plot.Align(plots, t, dc)
	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			if plots[j][i] != nil {
				plots[j][i].Draw(canvases[j][i])
			}
		}
	}

	w, err := os.Create("/tmp/" + filename + ".png")
	if err != nil {
		panic(err)
	}
	defer w.Close()
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(w); err != nil {
		panic(err)
	}
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

	p.Title.Text = ep.Description + " - " + strconv.Itoa(si) + "ms"

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for k, elem := range valuesMap {
		boxplot, err := plotter.NewBoxPlot(w, position, elem)
		errMgmt(err)
		nominals = append(nominals, strconv.Itoa(k)+"(M:"+strconv.Itoa(int(boxplot.Median))+")")
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

	p.Title.Text = ep.Description + " - " + strconv.Itoa(msgSize) + "KiB"

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for k, elem := range valuesMap {
		boxplot, err := plotter.NewBoxPlot(w, position, elem)
		errMgmt(err)
		nominals = append(nominals, strconv.Itoa(k)+"(M:"+strconv.Itoa(int(boxplot.Median))+")")
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

	p.Title.Text = strconv.Itoa(si) + "ms - " + strconv.Itoa(msgSize) + "KiB"

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for k, elem := range valuesMap {
		boxplot, err := plotter.NewBoxPlot(w, position, elem)
		errMgmt(err)
		nominals = append(nominals, k+"(M:"+strconv.Itoa(int(boxplot.Median))+")")
		mins = append(mins, boxplot.AdjLow)
		maxes = append(maxes, boxplot.AdjHigh)
		position += 1
		p.Add(boxplot)
	}
	p.NominalX(nominals...)

	return p, floats.Min(mins), floats.Max(maxes)
}

func nameFromDest(dest string, eps *[]struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}) (string, bool) {
	for _, b := range *eps {
		if b.Destination == dest {
			return b.Description, true
		}
	}
	return "", false
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func yValsCDF(length int) []float64 {
	var toReturn []float64
	for i := 0; i < length; i++ {
		toReturn = append(toReturn, float64(i)/float64(length-1))
	}
	return toReturn
}
