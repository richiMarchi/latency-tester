package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
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
	Intervals    []int  `yaml:"intervals"`     // in milliseconds
	MsgSizes     []int  `yaml:"msg_sizes"`     // in bytes
	ResponseSize int    `yaml:"response_size"` // in bytes
	TlsEnabled   string `yaml:"tls_enabled"`
	ExecDir      string `yaml:"exec_dir"`
}

const (
	ENDPOINTS = iota
	INTERVALS = iota
	SIZES     = iota
)

const AxisTicks = 15

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

	typedBoxPlots(settings, SIZES)
	typedBoxPlots(settings, INTERVALS)
	typedBoxPlots(settings, ENDPOINTS)
	typedCDFs(settings, SIZES)
	typedCDFs(settings, INTERVALS)
	typedCDFs(settings, ENDPOINTS)
	PingPlotter(settings)
	RttPlotter(settings)
}
