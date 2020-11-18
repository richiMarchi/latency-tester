package main

import (
	"encoding/csv"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

var endpoints = []string{
	"192.168.31.102:30011",
	"192.168.31.104:30011",
	"130.192.31.242:8080",
	"latency-tester.crownlabs.polito.it"}
var intervals = []int{10, 25, 50, 100, 250, 500}
var sizes = []int{1024, 10240, 102400, 1024000}

func main() {
	/*rows := len(endpoints)
	cols := len(intervals)
	plots := make([][]*plot.Plot, rows)
	for i := 0; i < rows; i++ {
		plots[i] = make([]*plot.Plot, cols)
		for j := 0; j < cols; j++ {
			// Boh
		}
	}*/
	p := singleBoxPlot("latency-tester.crownlabs.polito.it", 500)

	if err := p.Save(10*vg.Inch, 10*vg.Inch, "boxplot.png"); err != nil {
		panic(err)
	}
}

func singleBoxPlot(ep string, si int) *plot.Plot {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	// Open the desired files
	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), "-"+ep+".i") && strings.Contains(f.Name(), ".i"+strconv.Itoa(si)+".x") {
			file, err := os.Open(f.Name())
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

	w := vg.Points(150)
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
	b3, err := plotter.NewBoxPlot(w, 3, thousandK)
	if err != nil {
		panic(err)
	}
	p.Add(b0, b1, b2, b3)

	return p
}
