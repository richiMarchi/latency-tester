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
func nameFromDest(dest string, eps *[]EndpointData) (string, bool) {
	for _, b := range *eps {
		if strings.ReplaceAll(b.Destination, ":", "_") == dest {
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
		log.Fatal(LoggerHdr + err.Error())
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
func openDesiredFiles(execdir string, requestedRuns []int, nameLike ...string) []*os.File {
	files, err := ioutil.ReadDir(execdir + DataDirName)
	errMgmt(err)
	var openFiles []*os.File
	for _, f := range files {
		filename := filenameOnly(f.Name())
		if strings.Contains(filename, nameLike[0]) {
			fileRun, _ := strconv.Atoi(strings.Split(filename, "-")[0])
			// It can contain one or two strings, so it checks if the second value is present and then if it is in the name
			if len(nameLike) > 1 && !strings.Contains(filename, nameLike[1]) || !intInSlice(fileRun, requestedRuns) {
				continue
			}
			file, err := os.Open(execdir + DataDirName + f.Name())
			errMgmt(err)
			openFiles = append(openFiles, file)
		}
	}
	return openFiles
}

func closeOpenFiles(files []*os.File) {
	for _, f := range files {
		f.Close()
	}
}

// Return a BoxPlot graph and its min and max values with int as key of the map
func generateIntBoxPlotAndLimits(p *plot.Plot,
	valuesMap *map[int]plotter.Values,
	percentilesToRemove int,
	whiskerMin int,
	whiskerMax int) (*plot.Plot, float64, float64) {
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
		boxplot, err := plotter.NewBoxPlot(w, position, (*valuesMap)[k])
		errMgmt(err)
		handleWhiskerAndOutliers(boxplot, whiskerMin, whiskerMax, (*valuesMap)[k], toRemove)
		nominals = append(nominals, strconv.Itoa(k)+" (Median:"+strconv.FormatFloat(boxplot.Median, 'f', 2, 64)+")")
		mins = append(mins, (*valuesMap)[k][toRemove*percentilesToRemove])
		maxes = append(maxes, (*valuesMap)[k][len((*valuesMap)[k])-toRemove*percentilesToRemove-1])
		position += 1
		p.Add(boxplot)
	}
	if len(nominals) > 0 {
		p.NominalX(nominals...)
	} else {
		log.Fatal(LoggerHdr + "*** ERROR: Some files could be missing: plotting cannot be done")
	}

	return p, floats.Min(mins), floats.Max(maxes)
}

// Return a BoxPlot graph and its min and max values with string as key of the map
func generateStringBoxPlotAndLimits(p *plot.Plot,
	valuesMap *map[string]plotter.Values,
	percentilesToRemove int,
	whiskerMin int,
	whiskerMax int) (*plot.Plot, float64, float64) {
	// Get map ordered keys
	keys := make([]string, 0, len(*valuesMap))
	for k := range *valuesMap {
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
		sort.Float64s((*valuesMap)[k])
		toRemove := len((*valuesMap)[k]) / 100
		boxplot, err := plotter.NewBoxPlot(w, position, (*valuesMap)[k])
		errMgmt(err)
		handleWhiskerAndOutliers(boxplot, whiskerMin, whiskerMax, (*valuesMap)[k], toRemove)
		nominals = append(nominals, k+" (Median:"+strconv.FormatFloat(boxplot.Median, 'f', 2, 64)+")")
		mins = append(mins, (*valuesMap)[k][toRemove*percentilesToRemove])
		maxes = append(maxes, (*valuesMap)[k][len((*valuesMap)[k])-toRemove*percentilesToRemove-1])
		position += 1
		p.Add(boxplot)
	}
	if len(nominals) > 0 {
		p.NominalX(nominals...)
	} else {
		log.Fatal(LoggerHdr + "*** ERROR: Some files could be missing: plotting cannot be done")
	}

	return p, floats.Min(mins), floats.Max(maxes)
}

// Return a CDF graph with int as keys of the map
func generateIntCDFPlot(p *plot.Plot, valuesMap *map[int]plotter.Values, percentilesToRemove int) {
	// Get map ordered keys
	keys := make([]int, 0, len(*valuesMap))
	for k := range *valuesMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var lines []interface{}
	var mins []float64
	var maxes []float64
	for _, k := range keys {
		// Remove the last two percentiles in order to avoid unreadable plots
		sort.Float64s((*valuesMap)[k])
		toRemove := len((*valuesMap)[k]) / 100
		mins = append(mins, (*valuesMap)[k][toRemove*percentilesToRemove])
		maxes = append(maxes, (*valuesMap)[k][len((*valuesMap)[k])-toRemove*percentilesToRemove-1])
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len((*valuesMap)[k])) {
			toAdd = append(toAdd, plotter.XY{X: (*valuesMap)[k][i], Y: y})
		}
		lines = append(lines, strconv.Itoa(k))
		lines = append(lines, toAdd)
	}
	err := plotutil.AddLines(p, lines...)
	errMgmt(err)
	p.X.Min = floats.Min(mins)
	p.X.Max = floats.Max(maxes)
}

// Return a CDF graph with string as keys of the map
func generateStringCDFPlot(p *plot.Plot, valuesMap *map[string]plotter.Values, percentilesToRemove int) {
	// Get map ordered keys
	keys := make([]string, 0, len(*valuesMap))
	for k := range *valuesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []interface{}
	var mins []float64
	var maxes []float64
	for _, k := range keys {
		// Remove the last two percentiles in order to avoid unreadable plots
		sort.Float64s((*valuesMap)[k])
		toRemove := len((*valuesMap)[k]) / 100
		mins = append(mins, (*valuesMap)[k][toRemove*percentilesToRemove])
		maxes = append(maxes, (*valuesMap)[k][len((*valuesMap)[k])-toRemove*percentilesToRemove-1])
		var toAdd plotter.XYs
		for i, y := range yValsCDF(len((*valuesMap)[k])) {
			toAdd = append(toAdd, plotter.XY{X: (*valuesMap)[k][i], Y: y})
		}
		lines = append(lines, k)
		lines = append(lines, toAdd)
	}
	err := plotutil.AddLines(p, lines...)
	errMgmt(err)
	p.X.Min = floats.Min(mins)
	p.X.Max = floats.Max(maxes)
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
func getTcpPlotTitle(endpoints, intervals, sizes []string, streamCounter int) string {
	tracker := 0
	for _, addr := range endpoints {
		for _, inter := range intervals {
			for _, size := range sizes {
				if tracker == streamCounter {
					return "TCP ACK Latency: " + addr + " - " + inter + "ms - " + size + "B"
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
	return "Unavailable at the moment"
}

func requestedSlice(settings Settings) []int {
	if settings.RunsToPlot != nil {
		return settings.RunsToPlot
	}
	var toReturn []int
	for i := 1; i <= settings.Runs; i++ {
		toReturn = append(toReturn, i)
	}
	return toReturn
}

func rttValues(pairs plotter.XYs) []float64 {
	var rtts []float64
	for _, p := range pairs {
		rtts = append(rtts, p.Y)
	}
	return rtts
}

func filenameOnly(f string) string {
	return f[strings.LastIndex(f, "/")+1:]
}

func handleWhiskerAndOutliers(boxplot *plotter.BoxPlot,
	whiskerMin int,
	whiskerMax int,
	values plotter.Values,
	toRemove int) {
	var updatedOutliers []int
	if whiskerMin != 0 {
		boxplot.AdjLow = values[toRemove*whiskerMin]
		for counter := 0; counter < toRemove*whiskerMin; counter++ {
			updatedOutliers = append(updatedOutliers, counter)
		}
	} else {
		for counter, indexElem := range boxplot.Outside {
			if counter == 0 && indexElem != 0 {
				break
			}
			updatedOutliers = append(updatedOutliers, indexElem)
			if counter+1 < len(boxplot.Outside) && boxplot.Outside[counter+1] != (indexElem+1) {
				break
			}
		}
	}
	if whiskerMax != 0 {
		boxplot.AdjHigh = values[len(values)-toRemove*(100-whiskerMax)-1]
		for counter := len(values) - toRemove*(100-whiskerMax); counter < len(values); counter++ {
			updatedOutliers = append(updatedOutliers, counter)
		}
	} else {
		for counter, indexElem := range boxplot.Outside {
			if indexElem == 0 || counter != 0 && (indexElem-1) == boxplot.Outside[counter-1] {
				continue
			}
			for lateCounter := counter; lateCounter < len(boxplot.Outside); lateCounter++ {
				updatedOutliers = append(updatedOutliers, boxplot.Outside[lateCounter])
			}
			break
		}
	}
	boxplot.Outside = updatedOutliers
}

func configurePlotFontSizes(plot *plot.Plot, xTickSmall bool) {
	plot.Title.Font.Size = vg.Points(30)
	plot.X.Label.Font.Size = vg.Points(28)
	plot.Y.Label.Font.Size = vg.Points(28)
	plot.X.Tick.Label.Font.Size = vg.Points(26)
	if xTickSmall {
		plot.X.Tick.Label.Font.Size = vg.Points(14)
	}
	plot.Y.Tick.Label.Font.Size = vg.Points(26)
	plot.Legend.Font.Size = vg.Points(20)
}

func configurePlotFontSizesMultiple(plot *plot.Plot, xTickSmall bool) {
	plot.Title.Font.Size = vg.Points(24)
	plot.X.Label.Font.Size = vg.Points(22)
	plot.Y.Label.Font.Size = vg.Points(22)
	plot.X.Tick.Label.Font.Size = vg.Points(20)
	if xTickSmall {
		plot.X.Tick.Label.Font.Size = vg.Points(14)
	}
	plot.Y.Tick.Label.Font.Size = vg.Points(20)
	plot.Legend.Font.Size = vg.Points(20)
}
