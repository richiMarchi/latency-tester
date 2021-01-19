package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"syscall"
)

type Settings struct {
	Runs             int    `yaml:"runs"`
	RunsInterval     int    `yaml:"runs_interval"`      // in minutes
	RunsStepDuration int    `yaml:"runs_step_duration"` // in seconds
	IperfIp          string `yaml:"iperf_ip"`
	IperfPort        string `yaml:"iperf_port"`
	PingIp           string `yaml:"ping_ip"`
	PingInterval     int    `yaml:"ping_interval"` // in seconds
	Endpoints        []struct {
		Description string `yaml:"description"`
		Destination string `yaml:"destination"`
	} `yaml:"endpoints"`
	Intervals           []int  `yaml:"intervals"`     // in milliseconds
	MsgSizes            []int  `yaml:"msg_sizes"`     // in bytes
	ResponseSize        int    `yaml:"response_size"` // in bytes
	TlsEnabled          string `yaml:"tls_enabled"`
	ExecDir             string `yaml:"exec_dir"`
	PercentilesToRemove int    `yaml:"percentiles_to_remove"`
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
	if settings.IperfPort == "" {
		settings.IperfPort = "5201"
	}

	// Create the file in order that it can be totally handled by the host machine
	syscall.Umask(0)
	_ = os.Mkdir(settings.ExecDir+PlotDirName, os.ModePerm)

	var wg sync.WaitGroup

	wg.Add(8 + settings.Runs)
	for run := 1; run <= settings.Runs; run++ {
		go TcpdumpPlotter(settings, run, &wg)
	}
	go typedBoxPlots(settings, SIZES, &wg)
	go typedBoxPlots(settings, INTERVALS, &wg)
	go typedBoxPlots(settings, ENDPOINTS, &wg)
	go typedCDFs(settings, SIZES, &wg)
	go typedCDFs(settings, INTERVALS, &wg)
	go typedCDFs(settings, ENDPOINTS, &wg)
	go PingPlotter(settings, &wg)
	go RttPlotter(settings, &wg)
	wg.Wait()
}
