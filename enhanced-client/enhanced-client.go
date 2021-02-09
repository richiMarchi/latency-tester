package main

import (
	"bytes"
	"context"
	"github.com/lorenzosaino/go-sysctl"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
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
	Runs              int            `yaml:"runs"`
	RunsInterval      int            `yaml:"runs_interval"`      // in minutes
	RunsStepDuration  int            `yaml:"runs_step_duration"` // in seconds
	IperfDestinations []IperfData    `yaml:"iperf_destinations"`
	PingDestinations  []PingData     `yaml:"ping_destinations"`
	PingInterval      int            `yaml:"ping_interval"` // in seconds
	Endpoints         []EndpointData `yaml:"endpoints"`
	Intervals         []int          `yaml:"intervals"`     // in milliseconds
	MsgSizes          []int          `yaml:"msg_sizes"`     // in bytes
	ResponseSize      int            `yaml:"response_size"` // in bytes
	TcpdumpEnabled    bool           `yaml:"tcpdump_enabled"`
	TlsEnabled        string         `yaml:"tls_enabled"`
	ExecDir           string         `yaml:"exec_dir"`
}

func main() {

	if len(os.Args) == 1 {
		log.Fatal("Settings filename requested")
	}

	stopHealthChecker := make(chan os.Signal, 1)
	log.Println("Run Health Checker on '0.0.0.0:8080/health'")
	go runHealthChecker(stopHealthChecker)

	// Settings parsing
	file, err := ioutil.ReadFile(os.Args[1])
	errMgmt(err)
	var settings Settings
	log.Println("Reading settings")
	err = yaml.Unmarshal(file, &settings)
	errMgmt(err)

	if settings.RunsStepDuration == 0 && settings.RunsInterval == 0 {
		log.Fatal("One between runs_step_duration and runs_interval must be set")
	}
	combinations := len(settings.Endpoints) * len(settings.Intervals) * len(settings.MsgSizes)
	if settings.RunsStepDuration == 0 {
		settings.RunsStepDuration = settings.RunsInterval * 60 / combinations
		log.Println("Warning: runs_step_duration not set, the value will be", settings.RunsStepDuration)
	}
	if settings.RunsInterval == 0 {
		settings.RunsInterval = int(math.Ceil(float64(settings.RunsStepDuration*combinations) / 60))
		log.Println("Warning: runs_interval not set, the value will be", settings.RunsInterval)
	}
	avgSleep := float64(settings.RunsInterval*60-combinations*settings.RunsStepDuration) / 60
	if avgSleep < 0 {
		log.Println("Warning: the runs will be out of phase, not enough time to complete a run")
	} else {
		log.Println("Average sleep minutes between end and start of consecutive runs:", math.Round(avgSleep))
	}

	// Print flags statuses in order to be sure it is as expected
	ss, ssErr := sysctl.Get("net.ipv4.tcp_slow_start_after_idle")
	cc, ccErr := sysctl.Get("net.ipv4.tcp_congestion_control")
	if ssErr != nil || ccErr != nil {
		log.Println("Warning: Cannot access TCP system parameters.")
	} else {
		log.Println("TCP slow start after idle value: ", ss)
		log.Println("TCP congestion control algorithm: ", cc)
	}

	generateParamsFile(settings)

	// Start ping and tcpdump in background
	stopPing := make(chan os.Signal, len(settings.PingDestinations))
	var wg sync.WaitGroup
	wg.Add(len(settings.PingDestinations))
	killPing := false
	for _, dest := range settings.PingDestinations {
		log.Println("Starting ping thread towards", dest.Name)
		go pingThread(&wg, settings.ExecDir, dest, settings.PingInterval, stopPing, &killPing)
	}

	for i := 1; i <= settings.Runs; i++ {
		// Handle Iperf
		for _, iperfData := range settings.IperfDestinations {
			if iperfData.Port == "" {
				iperfData.Port = "5201"
			}
			log.Println("Running Iperf towards", iperfData.Name+"...")
			iperfer(i, settings.ExecDir, iperfData)
			log.Println("Iperf towards", iperfData.Name, "complete!")
		}
		// Handle Tcpdump
		stopTcpdump := make(chan os.Signal, 1)
		if settings.TcpdumpEnabled {
			wg.Add(1)
			localIp := getOutboundIP()
			log.Println("Starting Tcpdump")
			go tcpDumper(i, &wg, stopTcpdump, localIp, settings.ExecDir)
		}
		startTime := getTimestamp()
		// Start E2E analysis
		for _, addr := range settings.Endpoints {
			for _, inter := range settings.Intervals {
				for _, size := range settings.MsgSizes {
					repetitions := int((time.Duration(settings.RunsStepDuration) * time.Second).Milliseconds()) / inter
					log.Println("Run: " + strconv.Itoa(i) + " - " +
						"EP: " + addr.Destination + " - " +
						"Inter: " + strconv.Itoa(inter) + " - " +
						"Msg: " + strconv.Itoa(size))
					clientCmd := exec.Command("./client", "-reps="+strconv.Itoa(repetitions), "-interval="+strconv.Itoa(inter),
						"-requestPayload="+strconv.Itoa(size), "-responsePayload="+strconv.Itoa(settings.ResponseSize),
						"-tls="+settings.TlsEnabled, "-log="+settings.ExecDir+strconv.Itoa(i)+"-"+addr.Destination+
							".i"+strconv.Itoa(inter)+".x"+strconv.Itoa(size), addr.Destination)
					var stdErrClient bytes.Buffer
					clientCmd.Stderr = &stdErrClient
					err = clientCmd.Run()
					if err != nil {
						log.Println(err)
					}
					if stdErrClient.Len() > 0 {
						log.Println("*** CLIENT ERROR ***\n", stdErrClient.String())
					}
				}
			}
		}
		if settings.TcpdumpEnabled {
			log.Println("Signal Tcpdump Stop")
			stopTcpdump <- os.Interrupt
		}
		if i == settings.Runs {
			killPing = true
		}
		for range settings.PingDestinations {
			log.Println("Saving gathered ping data to file")
			stopPing <- os.Interrupt
		}
		if i != settings.Runs {
			elapsed := getTimestamp().Sub(startTime)
			waitTime := time.Duration(settings.RunsInterval)*time.Minute - elapsed
			if waitTime < 0 {
				log.Println("Warning: Run lasted more than 'run_interval'!")
			} else {
				log.Println("Sleeping for about", strconv.Itoa(int(waitTime.Seconds())), "seconds to wait until next run")
				time.Sleep(waitTime)
			}
		}
	}
	wg.Wait()

	// Plotting
	log.Println("Plotting...")
	plotterCmd := exec.Command("./plotter", os.Args[1])
	var stdErrPlotter bytes.Buffer
	plotterCmd.Stderr = &stdErrPlotter
	err = plotterCmd.Run()
	if err != nil {
		log.Println(err)
	}
	if stdErrPlotter.Len() > 0 {
		log.Println("*** PLOTTER ERROR ***\n", stdErrPlotter.String())
	}
	stopHealthChecker <- os.Interrupt
	log.Println("Everything's complete!")
}

func generateParamsFile(settings Settings) {
	log.Println("Generating parameters file")
	paramsFile, err := os.Create(settings.ExecDir + "parameters.txt")
	errMgmt(err)
	defer paramsFile.Close()

	destinations := ""
	for i, dest := range settings.Endpoints {
		if i != 0 {
			destinations += ","
		}
		destinations += dest.Description
	}
	paramsFile.WriteString(destinations + "\n")
	intervals := ""
	for i, inter := range settings.Intervals {
		if i != 0 {
			intervals += ","
		}
		intervals += strconv.Itoa(inter)
	}
	paramsFile.WriteString(intervals + "\n")
	sizes := ""
	for i, size := range settings.MsgSizes {
		if i != 0 {
			sizes += ","
		}
		sizes += strconv.Itoa(size)
	}
	paramsFile.WriteString(sizes)
}

func runHealthChecker(c chan os.Signal) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { return })
	server := http.Server{Addr: "0.0.0.0:8080"}

	// Handle stop
	go func() {
		for range c {
			log.Println("Stopping health checker")
			err := server.Shutdown(context.TODO())
			if err != nil {
				log.Println(err)
			}
		}
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func iperfer(run int, execdir string, iperfData IperfData) {
	iperfFile, err := os.Create(execdir + strconv.Itoa(run) + "-iperf_" + iperfData.Name + ".txt")
	errMgmt(err)
	defer iperfFile.Close()

	iperfRes, err := exec.Command(
		"timeout", "10", "iperf3", "-c", iperfData.Ip, "-p", iperfData.Port, "-t", "5").Output()
	_, err = iperfFile.Write(iperfRes)
	if err != nil {
		log.Println(err)
	}
}

func pingThread(wg *sync.WaitGroup, execdir string, destination PingData, interval int, c chan os.Signal, kill *bool) {
	osRtt, err := os.Create(execdir + "ping_" + destination.Name + ".txt")
	errMgmt(err)
	defer osRtt.Close()
	for !*kill {
		pingerCmd := exec.Command("ping", destination.Ip, "-i", strconv.Itoa(interval), "-D")

		// Handle stop
		go func() {
			for range c {
				log.Println("Stopping ping to save to file")
				err = pingerCmd.Process.Signal(os.Interrupt)
				if err != nil {
					log.Println(err)
				}
			}
		}()

		pingOutput, err := pingerCmd.Output()
		if err != nil {
			log.Println(err)
		}
		_, err = osRtt.Write(pingOutput)
		if err != nil {
			log.Println(err)
		}
	}
	wg.Done()
}

func tcpDumper(run int, wg *sync.WaitGroup, c chan os.Signal, localIp, execdir string) {
	tcpRtt, err := os.Create(execdir + strconv.Itoa(run) + "-tcpdump_report.csv")
	errMgmt(err)
	defer tcpRtt.Close()
	tcpRtt.WriteString("#frame-timestamp,tcp-ack-rtt,tcp-stream-id,retransmission\n")
	tcpdumpCmd := exec.Command("tshark",
		"-ni", "any",
		"-Y", "tcp.analysis.ack_rtt and ip.dst=="+localIp,
		"-e", "frame.time_epoch",
		"-e", "tcp.analysis.ack_rtt",
		"-e", "tcp.stream",
		"-e", "tcp.analysis.retransmission",
		"-T", "fields",
		"-E", "separator=,",
		"-E", "quote=d")

	// Handle stop
	go func() {
		for range c {
			log.Println("Stop Tcpdump")
			err = tcpdumpCmd.Process.Signal(os.Interrupt)
			if err != nil {
				log.Println(err)
			}
		}
	}()

	tcpdumpOutput, err := tcpdumpCmd.Output()
	if err != nil {
		log.Println(err)
	}
	_, err = tcpRtt.Write(tcpdumpOutput)
	if err != nil {
		log.Println(err)
	}
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

// Get preferred outbound ip of this machine
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
