package main

import (
	"encoding/csv"
	"fmt"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func Plot(settings Settings) {
	rows := len(settings.Endpoints)
	cols := len(settings.Intervals)
	var min float64 = 10000
	var max float64 = 0
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			var tmpMin float64
			var tmpMax float64
			plots[i][j], tmpMin, tmpMax = singleBoxPlot(
				settings.Endpoints[i].Destination, settings.Endpoints[i].Description, settings.Intervals[j], settings.MsgSizes)
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

	img := vgimg.New(vg.Points(3000), vg.Points(1000))
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

	w, err := os.Create("/tmp/boxplot.png")
	if err != nil {
		panic(err)
	}
	defer w.Close()
	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(w); err != nil {
		panic(err)
	}
}

func singleBoxPlot(ep_dest string, ep_name string, si int, msgSizes []int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for " + ep_name + " and send interval " + strconv.Itoa(si))
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	// Open the desired files
	files, err := ioutil.ReadDir("/tmp")
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), "-"+ep_dest+".i") && strings.Contains(f.Name(), ".i"+strconv.Itoa(si)+".x") {
			file, err := os.Open("/tmp/" + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			openFiles = append(openFiles, file)
		}
	}

	valuesMap := make(map[int]plotter.Values)

	for _, f := range openFiles {
		records, _ := csv.NewReader(f).ReadAll()
		for i, row := range records {
			if i != 0 {
				parsed, fail := strconv.ParseFloat(row[2], 64)
				if fail != nil {
					continue
				}
				parsedSizeVal, err := strconv.ParseInt(f.Name()[strings.LastIndex(f.Name(), "x")+1:len(f.Name())-4], 10, 32)
				sizeVal := int(parsedSizeVal)
				errMgmt(err)
				if intInSlice(sizeVal, msgSizes) {
					valuesMap[sizeVal] = append(valuesMap[sizeVal], parsed)
				}
			}
		}
	}

	p.X.Label.Text = "Request Size (KiB)"
	p.Y.Label.Text = "E2E RTT (ms)"

	p.Title.Text = ep_name + " - " + strconv.Itoa(si) + "ms"

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for k, elem := range valuesMap {
		nominals = append(nominals, strconv.Itoa(k))
		boxplot, err := plotter.NewBoxPlot(w, position, elem)
		errMgmt(err)
		mins = append(mins, boxplot.AdjLow)
		maxes = append(maxes, boxplot.AdjHigh)
		position += 1
		p.Add(boxplot)
	}
	p.NominalX(nominals...)

	return p, floats.Min(mins), floats.Max(maxes)
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
