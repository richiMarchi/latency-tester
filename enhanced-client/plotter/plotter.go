package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"syscall"
)

type IperfData struct {
	Name string `yaml:"name"`
	Ip   string `yaml:"ip"`
	Port string `yaml:"port"`
}

type PingData struct {
	Name string `yaml:"name"`
	Ip   string `yaml:"ip"`
}

type EndpointData struct {
	Description string `yaml:"description"`
	Destination string `yaml:"destination"`
}

type Settings struct {
	Runs                int            `yaml:"runs"`
	RunsInterval        int            `yaml:"runs_interval"`      // in minutes
	RunsStepDuration    int            `yaml:"runs_step_duration"` // in seconds
	IperfDestinations   []IperfData    `yaml:"iperf_data"`
	PingDestinations    []PingData     `yaml:"ping_destinations"`
	PingInterval        int            `yaml:"ping_interval"` // in seconds
	Endpoints           []EndpointData `yaml:"endpoints"`
	Intervals           []int          `yaml:"intervals"`     // in milliseconds
	MsgSizes            []int          `yaml:"msg_sizes"`     // in bytes
	ResponseSize        int            `yaml:"response_size"` // in bytes
	TlsEnabled          string         `yaml:"tls_enabled"`
	ExecDir             string         `yaml:"exec_dir"`
	PercentilesToRemove int            `yaml:"percentiles_to_remove"`
	RttMin              float64        `yaml:"rtt_min"`
	RttMax              float64        `yaml:"rtt_max"`
	RunsToPlot          []int          `yaml:"runs_to_plot"`
}

const (
	ENDPOINTS = iota
	INTERVALS = iota
	SIZES     = iota
)

const AxisTicks = 15
const PlotDirName = "/plots/"

func main() {

	if len(os.Args) == 1 {
		log.Fatal("Settings filename requested")
	}

	// Settings parsing
	file, err := ioutil.ReadFile(os.Args[1])
	errMgmt(err)
	var settings Settings
	err = yaml.Unmarshal(file, &settings)
	errMgmt(err)

	// Create the file in order that it can be totally handled by the host machine
	syscall.Umask(0)
	_ = os.Mkdir(settings.ExecDir+PlotDirName, os.ModePerm)

	var wg sync.WaitGroup

	wg.Add(8 + len(requestedSlice(settings)))
	for _, run := range requestedSlice(settings) {
		go TcpdumpPlotter(settings, run, &wg)
	}
	go typedBoxPlots(settings, SIZES, &wg)
	go typedBoxPlots(settings, INTERVALS, &wg)
	go typedBoxPlots(settings, ENDPOINTS, &wg)
	go typedCDFs(settings, SIZES, &wg)
	go typedCDFs(settings, INTERVALS, &wg)
	go typedCDFs(settings, ENDPOINTS, &wg)
	go PingPlotter(settings, &wg)
	// Generates 2 pdfs, standard and boxplots
	go RttPlotter(settings, &wg)
	wg.Wait()
}
