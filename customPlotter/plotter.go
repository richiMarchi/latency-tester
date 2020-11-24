package customPlotter

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

func Plot(endpoints []string, intervals []int) {
	rows := len(endpoints)
	cols := len(intervals)
	var min float64 = 10000
	var max float64 = 0
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			var tmpMin float64
			var tmpMax float64
			plots[i][j], tmpMin, tmpMax = singleBoxPlot(endpoints[i], intervals[j])
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

func singleBoxPlot(ep string, si int) (*plot.Plot, float64, float64) {
	fmt.Println("Plot for " + ep + " and send interval " + strconv.Itoa(si))
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
		if strings.Contains(f.Name(), "-"+ep+".i") && strings.Contains(f.Name(), ".i"+strconv.Itoa(si)+".x") {
			file, err := os.Open("/tmp/" + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			openFiles = append(openFiles, file)
		}
	}

	var oneK plotter.Values
	var tenK plotter.Values
	var hundredK plotter.Values
	var thousandK plotter.Values

	for _, f := range openFiles {
		records, _ := csv.NewReader(f).ReadAll()
		for i, row := range records {
			if i != 0 {
				parsed, fail := strconv.ParseFloat(row[2], 64)
				if fail != nil {
					continue
				}
				if strings.Contains(f.Name(), "x1024000.csv") {
					thousandK = append(thousandK, parsed)
				} else if strings.Contains(f.Name(), "x102400.csv") {
					hundredK = append(hundredK, parsed)
				} else if strings.Contains(f.Name(), "x10240.csv") {
					tenK = append(tenK, parsed)
				} else if strings.Contains(f.Name(), "x1024.csv") {
					oneK = append(oneK, parsed)
				} else {
					panic("Cannot get the right size of this file: " + f.Name())
				}
			}
		}
	}

	p.X.Label.Text = "Request Size (KiB)"
	p.Y.Label.Text = "E2E RTT (ms)"
	p.NominalX("1", "10", "100", "1000")
	p.Title.Text = ep + " - " + strconv.Itoa(si) + "ms"
	//p.Y.Min = 0

	w := vg.Points(100)
	b0, err := plotter.NewBoxPlot(w, 0, oneK)
	if err != nil {
		panic(err)
	}
	b1, err := plotter.NewBoxPlot(w, 1, tenK)
	if err != nil {
		panic(err)
	}
	b2, err := plotter.NewBoxPlot(w, 2, hundredK)
	if err != nil {
		panic(err)
	}
	/*b3, err := plotter.NewBoxPlot(w, 3, thousandK)
	if err != nil {
		panic(err)
	}*/

	p.Add(b0, b1, b2 /*, b3*/)
	//p.Y.Min = floats.Min([]float64{b0.AdjLow, b1.AdjLow, b2.AdjLow /*, b3.AdjLow*/}) - 1
	//p.Y.Max = floats.Max([]float64{b0.AdjHigh, b1.AdjHigh, b2.AdjHigh /*, b3.AdjHigh*/}) + 10

	return p, floats.Min([]float64{b0.AdjLow, b1.AdjLow, b2.AdjLow /*, b3.AdjLow*/}),
		floats.Max([]float64{b0.AdjHigh, b1.AdjHigh, b2.AdjHigh /*, b3.AdjHigh*/})
}
