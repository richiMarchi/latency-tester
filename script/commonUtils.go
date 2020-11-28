package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"log"
	"os"
	"time"
)

func commonPlotting(plots [][]*plot.Plot, rows int, cols int, cardWidth int, filename string) {

	img := vgimg.New(vg.Points(float64(cardWidth)), vg.Points(float64(rows*650)))
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

func getTimestamp() time.Time {
	return time.Now()
}

func errMgmt(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ackedPacketInSlice(ack uint32, list *[]gopacket.Packet) gopacket.Packet {
	for i, b := range *list {
		if b.Layer(layers.LayerTypeTCP).(*layers.TCP).Seq == ack-1 {
			(*list)[i] = (*list)[len(*list)-1]
			*list = (*list)[:len(*list)-1]
			return b
		}
	}
	return nil
}
