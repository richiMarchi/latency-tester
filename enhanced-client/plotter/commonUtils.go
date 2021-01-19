package main

import (
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgpdf"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Draw a matrix of plots into a PDF
func commonPlotting(plots [][]*plot.Plot, rows int, cols int, cardWidth int, filename string) {

	img := vgpdf.New(vg.Points(float64(cardWidth)), vg.Points(float64(rows*650)))
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
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if plots[i][j] != nil {
				plots[i][j].Draw(canvases[i][j])
			}
		}
	}

	w, err := os.Create(filename + ".pdf")
	if err != nil {
		panic(err)
	}
	defer w.Close()
	if _, err := img.WriteTo(w); err != nil {
		panic(err)
	}
}

// Return the name of a destination given the address
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

// True if the int value is in the slice
func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Return the Y axis values for the CDF graph
func yValsCDF(length int) []float64 {
	var toReturn []float64
	for i := 0; i < length; i++ {
		toReturn = append(toReturn, float64(i)/float64(length-1))
	}
	return toReturn
}

func errMgmt(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Unifies the min and max of each plot Y axis in the matrix
func adjustMinMaxY(plots [][]*plot.Plot, rows, cols int, min, max float64) {
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].Y.Min = min
			plots[i][j].Y.Max = max
		}
	}
}

// Unifies the min and max of each plot X axis in the matrix
func adjustMinMaxX(plots [][]*plot.Plot, rows, cols int, min, max float64) {
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			plots[i][j].X.Min = min
			plots[i][j].X.Max = max
		}
	}
}

// Open the files with the name containing the nameLike strings
func openDesiredFiles(execdir string, nameLike ...string) []*os.File {
	files, err := ioutil.ReadDir(execdir)
	if err != nil {
		log.Fatal(err)
	}
	var openFiles []*os.File
	for _, f := range files {
		if strings.Contains(f.Name(), nameLike[0]) {
			// It can contain one or two strings, so it checks if the second value is present and then if it is in the name
			if len(nameLike) > 1 && !strings.Contains(f.Name(), nameLike[1]) {
				continue
			}
			file, err := os.Open(execdir + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			openFiles = append(openFiles, file)
		}
	}
	return openFiles
}

// Return a BoxPlot graph and its min and max values
func generateBoxPlotAndLimits(p *plot.Plot,
	valuesMap *map[int]plotter.Values,
	percentilesToRemove int) (*plot.Plot, float64, float64) {
	// Get map ordered keys
	keys := make([]int, 0, len(*valuesMap))
	for k := range *valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var nominals []string
	var mins []float64
	var maxes []float64
	w := vg.Points(100)
	var position float64 = 0
	for _, k := range keys {
		// Remove the first three and last three percentiles in order to avoid unreadable plots
		sort.Float64s((*valuesMap)[k])
		toRemove := len((*valuesMap)[k]) / 100
		(*valuesMap)[k] = (*valuesMap)[k][toRemove*percentilesToRemove : len((*valuesMap)[k])-toRemove*percentilesToRemove]
		boxplot, err := plotter.NewBoxPlot(w, position, (*valuesMap)[k])
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

// Return a CDF graph
func generateCDFPlot(p *plot.Plot, valuesMap *map[int]plotter.Values, percentilesToRemove int) {
	// Get map ordered keys
	keys := make([]int, 0, len(*valuesMap))
	for k := range *valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var lines []interface{}
	for _, k := range keys {
		// Remove the last two percentiles in order to avoid unreadable plots
		sort.Float64s((*valuesMap)[k])
		toRemove := len((*valuesMap)[k]) / 100
		(*valuesMap)[k] = (*valuesMap)[k][:len((*valuesMap)[k])-toRemove*percentilesToRemove]
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len((*valuesMap)[k])) {
			toAdd = append(toAdd, plotter.XY{X: (*valuesMap)[k][i], Y: y})
		}
		lines = append(lines, strconv.Itoa(k))
		lines = append(lines, toAdd)
	}
	err := plotutil.AddLines(p, lines...)
	errMgmt(err)
}

// Return the lengths of the elements depending on the object type
func getLoopElems(settings Settings, objectType int) (int, int, int) {
	switch objectType {
	case ENDPOINTS:
		return len(settings.MsgSizes), len(settings.Intervals), len(settings.Endpoints)
	case INTERVALS:
		return len(settings.Endpoints), len(settings.MsgSizes), len(settings.Intervals)
	case SIZES:
		return len(settings.Endpoints), len(settings.Intervals), len(settings.MsgSizes)
	default:
		panic("Wrong objectType in loop elements: only values 0,1 and 2 are allowed")
	}
}

// Return the title to assign to the plot depending on the stream
func getTcpPlotTitle(settings Settings, streamCounter int) string {
	tracker := 0
	for _, addr := range settings.Endpoints {
		for _, inter := range settings.Intervals {
			for _, size := range settings.MsgSizes {
				if tracker == streamCounter {
					return "TCP ACK Latency: " + addr.Description + " - " + strconv.Itoa(inter) + "ms - " + strconv.Itoa(size) + "B"
				}
				tracker += 1
				if tracker > streamCounter {
					break
				}
			}
			if tracker > streamCounter {
				break
			}
		}
		if tracker > streamCounter {
			break
		}
	}
	panic("Cannot assign the title to the plot, buggy code!")
}
