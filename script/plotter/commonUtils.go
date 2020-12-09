package main

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgpdf"
	"log"
	"os"
)

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

	w, err := os.Create(LogPath + filename + ".pdf")
	if err != nil {
		panic(err)
	}
	defer w.Close()
	if _, err := img.WriteTo(w); err != nil {
		panic(err)
	}
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

func errMgmt(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
