package main

import (
	"github.com/lorenzosaino/go-sysctl"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
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
}

func main() {

	if len(os.Args) == 1 {
		log.Fatal("Settings filename requested")
	}

	// Settings parsing
	file, err := ioutil.ReadFile("/tmp/" + os.Args[1])
	errMgmt(err)
	var settings Settings
	err = yaml.Unmarshal(file, &settings)
	errMgmt(err)
	if settings.IperfPort == "" {
		settings.IperfPort = "5201"
	}

	// Print flags statuses in order to be sure it is as expected
	val, err := sysctl.Get("net.ipv4.tcp_slow_start_after_idle")
	errMgmt(err)
	log.Println("TCP slow start after idle value: ", val)
	val, err = sysctl.Get("net.ipv4.tcp_congestion_control")
	errMgmt(err)
	log.Println("TCP congestion control algorithm: ", val)

	// Start ping and tcpdump in background
	c := make(chan os.Signal, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go pingThread(&wg, settings.PingIp, settings.PingInterval, c)
	go tcpDumper(&wg, c)

	for i := 1; i <= settings.Runs; i++ {
		log.Println("Running Iperf...")
		iperfer(i, settings.IperfIp, settings.IperfPort)
		log.Println("Iperf complete!")
		startTime := getTimestamp()
		for _, addr := range settings.Endpoints {
			for _, inter := range settings.Intervals {
				for _, size := range settings.MsgSizes {
					repetitions := int((time.Duration(settings.RunsStepDuration) * time.Second).Milliseconds()) / inter
					log.Println("Run: " + strconv.Itoa(i) + " - " +
						"EP: " + addr.Destination + " - " +
						"Inter: " + strconv.Itoa(inter) + " - " +
						"Msg: " + strconv.Itoa(size))
					err = exec.Command("./client", "-reps="+strconv.Itoa(repetitions), "-interval="+strconv.Itoa(inter),
						"-requestPayload="+strconv.Itoa(size), "-responsePayload="+strconv.Itoa(settings.ResponseSize),
						"-tls="+settings.TlsEnabled, "-log="+strconv.Itoa(i)+"-"+addr.Destination+".i"+strconv.Itoa(inter)+".x"+
							strconv.Itoa(size), addr.Destination).Run()
					errMgmt(err)
				}
			}
		}
		if i != settings.Runs {
			elapsed := getTimestamp().Sub(startTime)
			waitTime := time.Duration(settings.RunsInterval)*time.Minute - elapsed
			if waitTime < 0 {
				log.Println("Warning: Run lasted more than 'run_interval'!")
			} else {
				time.Sleep(waitTime)
			}
		}
	}

	c <- os.Interrupt
	c <- os.Interrupt
	wg.Wait()

	// Plotting
	log.Println("Plotting...")
	Plot(settings)
	errMgmt(err)
	log.Println("Everything's complete!")
}

func iperfer(run int, ip string, port string) {
	iperfFile, err := os.Create("/tmp/" + strconv.Itoa(run) + "-iperf_report.txt")
	errMgmt(err)
	defer iperfFile.Close()

	iperfRes, err := exec.Command("iperf3", "-c", ip, "-p", port, "-t", "5").Output()
	errMgmt(err)
	_, err = iperfFile.Write(iperfRes)
	errMgmt(err)
}

func pingThread(wg *sync.WaitGroup, address string, interval int, c chan os.Signal) {
	osRtt, err := os.Create("/tmp/ping_report.txt")
	errMgmt(err)
	defer osRtt.Close()
	pingerCmd := exec.Command("ping", address, "-i", strconv.Itoa(interval), "-D")

	// Handle stop
	go func() {
		for range c {
			_ = pingerCmd.Process.Signal(os.Interrupt)
		}
	}()

	pingOutput, err := pingerCmd.Output()
	errMgmt(err)
	_, err = osRtt.Write(pingOutput)
	errMgmt(err)
	wg.Done()
}

func tcpDumper(wg *sync.WaitGroup, c chan os.Signal) {
	tcpdumper := exec.Command("tcpdump", "-U", "-s", "96", "-w", "/tmp/tcpdump_report.pcap")

	// Handle stop
	go func() {
		for range c {
			_ = tcpdumper.Process.Signal(os.Interrupt)
		}
	}()

	err := tcpdumper.Run()
	errMgmt(err)
	wg.Done()
}

func getTimestamp() time.Time {
	return time.Now()
}

func errMgmt(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
