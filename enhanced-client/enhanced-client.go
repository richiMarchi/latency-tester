package main

import (
	"bytes"
	"context"
	"github.com/lorenzosaino/go-sysctl"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	TlsEnabled        bool           `yaml:"tls_enabled"`
	ExecDir           string         `yaml:"exec_dir"`
}

const DataDirName = "raw-data/"

func main() {

	const LoggerHdr = "@main          - "

	if len(os.Args) == 1 {
		log.Fatal(LoggerHdr + "Settings filename requested")
	}

	stopHealthChecker := make(chan os.Signal, 1)
	log.Println(LoggerHdr + "Run Health Checker on '0.0.0.0:8080/health'")
	go healthChecker(stopHealthChecker)

	// Settings parsing
	log.Println(LoggerHdr + "Opening settings file")
	file, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR reading settings file:", err)
	} else {
		log.Println(LoggerHdr + "Settings file successfully read")
	}
	var settings Settings
	log.Println(LoggerHdr + "Reading settings")
	err = yaml.Unmarshal(file, &settings)
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR reading settings:", err)
	} else {
		log.Println(LoggerHdr + "Settings successfully read")
	}

	if settings.RunsStepDuration == 0 && settings.RunsInterval == 0 {
		log.Fatal(LoggerHdr + "One between runs_step_duration and runs_interval must be set")
	}
	combinations := len(settings.Endpoints) * len(settings.Intervals) * len(settings.MsgSizes)
	if settings.RunsStepDuration == 0 {
		settings.RunsStepDuration = settings.RunsInterval * 60 / combinations
		log.Println(LoggerHdr+"WARNING: runs_step_duration not set, the value will be", settings.RunsStepDuration)
	}
	if settings.RunsInterval == 0 {
		settings.RunsInterval = int(math.Ceil(float64(settings.RunsStepDuration*combinations) / 60))
		log.Println(LoggerHdr+"WARNING: runs_interval not set, the value will be", settings.RunsInterval)
	}
	avgSleep := float64(settings.RunsInterval*60-combinations*settings.RunsStepDuration) / 60
	if avgSleep < 0 {
		log.Println(LoggerHdr + "WARNING: the runs will be out of phase, not enough time to complete a run")
	} else {
		log.Println(LoggerHdr+"Average sleep minutes between end and start of consecutive runs:",
			math.Round(avgSleep))
	}

	// Print flags statuses in order to be sure it is as expected
	ss, ssErr := sysctl.Get("net.ipv4.tcp_slow_start_after_idle")
	cc, ccErr := sysctl.Get("net.ipv4.tcp_congestion_control")
	if ssErr != nil || ccErr != nil {
		log.Println(LoggerHdr + "WARNING: Cannot access TCP system parameters.")
	} else {
		log.Println(LoggerHdr+"TCP slow start after idle value: ", ss)
		log.Println(LoggerHdr+"TCP congestion control algorithm: ", cc)
	}

	ts := getTimestamp()
	year, month, day := ts.UTC().Date()
	hour := ts.Hour()
	min := ts.Minute()
	sec := ts.Second()
	folderName := strconv.Itoa(year) + "-" + month.String() + "-" + strconv.Itoa(day) + "_" + strconv.Itoa(hour) + "-" +
		strconv.Itoa(min) + "-" + strconv.Itoa(sec)

	settings.ExecDir += folderName + "/"
	log.Println(LoggerHdr+"Creating folder named", folderName)
	err = os.Mkdir(settings.ExecDir, os.ModePerm)
	if err != nil {
		log.Println(LoggerHdr + err.Error())
	}
	log.Println(LoggerHdr + "Copying settings file in the results folder")
	fromFile, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR opening settings file:", err)
	} else {
		log.Println(LoggerHdr + "Settings file successfully opened")
	}
	toFile, err := os.Create(settings.ExecDir + fromFile.Name()[strings.LastIndex(fromFile.Name(), "/")+1:])
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR creating results folder settings file:", err)
	} else {
		log.Println(LoggerHdr + "Results folder settings file successfully created")
	}
	_, err = io.Copy(toFile, fromFile)
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR copying settings file into results folder:", err)
	} else {
		log.Println(LoggerHdr + "Settings file successfully copied into results folder")
	}
	// Create the file in order that it can be totally handled by the host machine
	log.Println(LoggerHdr + "Creating 'plots' folder")
	err = os.Mkdir(settings.ExecDir+DataDirName, os.ModePerm)
	if err != nil {
		log.Println(LoggerHdr + err.Error())
	}

	genParamsFile(settings)

	// Start ping and tcpdump in background
	stopPing := make(chan os.Signal, len(settings.PingDestinations))
	var wg sync.WaitGroup
	wg.Add(len(settings.PingDestinations))
	killPing := false
	for _, dest := range settings.PingDestinations {
		go pingThread(&wg, settings.ExecDir+DataDirName, dest, settings.PingInterval, stopPing, &killPing)
	}

	for i := 1; i <= settings.Runs; i++ {
		// Handle Iperf
		for _, iperfData := range settings.IperfDestinations {
			if iperfData.Port == "" {
				iperfData.Port = "5201"
			}
			iperfer(i, settings.ExecDir+DataDirName, iperfData)
		}
		// Handle Tcpdump
		stopTcpdump := make(chan os.Signal, 1)
		if settings.TcpdumpEnabled {
			log.Println(LoggerHdr + "Tcpdump is requested")
			wg.Add(1)
			localIp := getOutboundIP()
			log.Println(LoggerHdr + "Outbound IP: " + localIp)
			go tcpDumper(i, &wg, stopTcpdump, localIp, settings.ExecDir+DataDirName)
			time.Sleep(time.Second)
		} else {
			log.Println(LoggerHdr + "Tcpdump is not requested")
		}
		startTime := getTimestamp()
		// Start E2E analysis
		for _, addr := range settings.Endpoints {
			for _, inter := range settings.Intervals {
				for _, size := range settings.MsgSizes {
					repetitions := int((time.Duration(settings.RunsStepDuration) * time.Second).Milliseconds()) / inter
					log.Println(LoggerHdr + "Run: " + strconv.Itoa(i) + " - " +
						"EP: " + addr.Destination + " - " +
						"Inter: " + strconv.Itoa(inter) + " - " +
						"Msg: " + strconv.Itoa(size))
					clientCmd := exec.Command("./client",
						"-reps="+strconv.Itoa(repetitions),
						"-interval="+strconv.Itoa(inter),
						"-requestPayload="+strconv.Itoa(size),
						"-responsePayload="+strconv.Itoa(settings.ResponseSize),
						"-tls="+strconv.FormatBool(settings.TlsEnabled),
						"-log="+settings.ExecDir+DataDirName+strconv.Itoa(i)+"-"+strings.ReplaceAll(addr.Destination, ":", "_")+
							".i"+strconv.Itoa(inter)+".x"+strconv.Itoa(size),
						addr.Destination)
					var stdErrClient bytes.Buffer
					clientCmd.Stderr = &stdErrClient
					err = clientCmd.Run()
					if err != nil {
						log.Println(LoggerHdr+"*** ERROR executing client:", err)
					} else {
						log.Println(LoggerHdr + "OK! - Client executed successfully")
					}
					if stdErrClient.Len() > 0 {
						log.Println(LoggerHdr+"*** CLIENT STDERR ***\n", stdErrClient.String())
					}
				}
			}
		}
		if settings.TcpdumpEnabled {
			log.Println(LoggerHdr + "Signal Tcpdump Stop")
			stopTcpdump <- os.Interrupt
			log.Println(LoggerHdr + "Signal to stop tcpdump successfully received")
		}
		if i == settings.Runs {
			killPing = true
		}
		for _, pingDst := range settings.PingDestinations {
			log.Println(LoggerHdr + "Signal " + pingDst.Name + " ping to store gathered data to file")
			stopPing <- os.Interrupt
			log.Println(LoggerHdr + "Interrupt to store " + pingDst.Name + " ping data to file successfully received")
		}
		if i != settings.Runs {
			elapsed := getTimestamp().Sub(startTime)
			waitTime := time.Duration(settings.RunsInterval)*time.Minute - elapsed
			if waitTime < 0 {
				log.Println(LoggerHdr + "WARNING: Run lasted more than 'run_interval' of about " +
					strconv.FormatFloat(math.Abs(waitTime.Seconds()), 'f', 2, 64) + " seconds")
			} else {
				log.Println(LoggerHdr+"Sleeping for about", strconv.Itoa(int(waitTime.Seconds())),
					"seconds to wait until next run")
				time.Sleep(waitTime)
			}
		}
	}
	wg.Wait()

	// Plotting
	log.Println(LoggerHdr + "Plotting...")
	plotterCmd := exec.Command("./plotter", "-dir="+folderName, os.Args[1])
	var stdErrPlotter bytes.Buffer
	plotterCmd.Stderr = &stdErrPlotter
	err = plotterCmd.Run()
	if err != nil {
		log.Println(LoggerHdr+"*** ERROR plotting data:", err)
	} else {
		log.Println(LoggerHdr + "Plotting successfully executed")
	}
	log.Print(LoggerHdr + "Plotting Logs:\n" + stdErrPlotter.String())
	stopHealthChecker <- os.Interrupt
	log.Println(LoggerHdr + "Everything's complete!")
}

func genParamsFile(settings Settings) {
	const LoggerHdr = "@genParamsFile - "
	log.Println(LoggerHdr + "Generating parameters file")
	paramsFile, err := os.Create(settings.ExecDir + "parameters.txt")
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR creating parameters file:", err)
	} else {
		log.Println(LoggerHdr + "Parameters file successfully created")
	}
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

func healthChecker(c chan os.Signal) {
	const LoggerHdr = "@healthChecker - "

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { return })
	server := http.Server{Addr: "0.0.0.0:8080"}

	// Handle stop
	go func() {
		for range c {
			log.Println(LoggerHdr + "Stopping health checker")
			err := server.Shutdown(context.TODO())
			if err != nil {
				log.Println(LoggerHdr+"*** ERROR stopping health checker:", err)
			} else {
				log.Println(LoggerHdr + "Health checker successfully stopped")
			}
			break
		}
	}()

	log.Println(LoggerHdr + "Health checker listening...")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(LoggerHdr+"*** ERROR in ListenAndServe():", err)
	}
}

func iperfer(run int, execdir string, iperfData IperfData) {
	const LoggerHdr = "@iperfer       - "

	log.Println(LoggerHdr + "Creating " + iperfData.Name + " iperf file")
	iperfFile, err := os.Create(execdir + strconv.Itoa(run) + "-iperf_" + iperfData.Name + ".txt")
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR creating "+iperfData.Name+" iperf file:", err)
	} else {
		log.Println(LoggerHdr + iperfData.Name + " iperf file successfully created")
	}
	defer iperfFile.Close()

	log.Println(LoggerHdr+"Running Iperf towards", iperfData.Name+"...")
	iperfRes, err := exec.Command(
		"timeout", "10", "iperf3", "-c", iperfData.Ip, "-p", iperfData.Port, "-t", "5").Output()
	if err != nil {
		log.Println(LoggerHdr+"*** ERROR running iperf towards "+iperfData.Name+":", err)
	} else {
		log.Println(LoggerHdr+"Iperf towards", iperfData.Name, "complete!")
	}
	log.Println(LoggerHdr + "Saving " + iperfData.Name + " iperf output to file")
	_, err = iperfFile.Write(iperfRes)
	if err != nil {
		log.Println(LoggerHdr+"*** ERROR writing "+iperfData.Name+" iperf output to file", err)
	} else {
		log.Println(LoggerHdr + iperfData.Name + " iperf output successfully saved to file")
	}
}

func pingThread(wg *sync.WaitGroup, execdir string, destination PingData, interval int, c chan os.Signal, kill *bool) {
	const LoggerHdr = "@pingThread    - "

	log.Println(LoggerHdr + "Creating " + destination.Name + " ping file")
	osRtt, err := os.Create(execdir + "ping_" + destination.Name + ".txt")
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR creating "+destination.Name+" ping file:", err)
	} else {
		log.Println(LoggerHdr + destination.Name + " ping file successfully created")
	}
	defer osRtt.Close()
	for !*kill {
		pingerCmd := exec.Command("ping", destination.Ip, "-i", strconv.Itoa(interval), "-D")

		// Handle stop
		go func() {
			for range c {
				log.Println(LoggerHdr + "Stopping " + destination.Name + " ping to save to file")
				err = pingerCmd.Process.Signal(os.Interrupt)
				if err != nil {
					log.Println(LoggerHdr+"*** ERROR stopping "+destination.Name+" ping:", err)
				} else {
					log.Println(LoggerHdr + destination.Name + " ping successfully stopped")
				}
				break
			}
		}()

		log.Println(LoggerHdr+"Starting ping towards", destination.Name)
		pingOutput, err := pingerCmd.Output()
		if err != nil {
			log.Fatal(LoggerHdr+"*** ERROR pinging "+destination.Name+":", err)
		} else {
			log.Println(LoggerHdr + destination.Name + " ping successfully executed")
		}
		log.Println(LoggerHdr + "Saving to file " + destination.Name + " ping output")
		_, err = osRtt.Write(pingOutput)
		if err != nil {
			log.Println(LoggerHdr+"*** ERROR saving "+destination.Name+" ping output to file:", err)
		} else {
			log.Println(LoggerHdr + destination.Name + " ping data successfully saved to file")
		}
	}
	wg.Done()
}

func tcpDumper(run int, wg *sync.WaitGroup, c chan os.Signal, localIp, execdir string) {
	const LoggerHdr = "@tcpDumper     - "

	log.Println(LoggerHdr + "Creating tcpdump output file for run " + strconv.Itoa(run))
	tcpRtt, err := os.Create(execdir + strconv.Itoa(run) + "-tcpdump_report.csv")
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR creating tcpdump output file for run "+strconv.Itoa(run), err)
	} else {
		log.Println(LoggerHdr + "Tcpdump output file for run " + strconv.Itoa(run) + " successfully created")
	}
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
			log.Println(LoggerHdr + "Stopping tcpdump for run " + strconv.Itoa(run))
			err = tcpdumpCmd.Process.Signal(os.Interrupt)
			if err != nil {
				log.Println(LoggerHdr+"*** ERROR stopping tcpdump for run "+strconv.Itoa(run)+":", err)
			} else {
				log.Println(LoggerHdr + "Tcpdump for run " + strconv.Itoa(run) + " successfully stopped")
			}
			break
		}
	}()

	log.Println(LoggerHdr + "Starting tcpdump for run " + strconv.Itoa(run))
	tcpdumpOutput, err := tcpdumpCmd.Output()
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR executing tcpdump for run "+strconv.Itoa(run)+":", err)
	} else {
		log.Println(LoggerHdr + "Tcpdump for run " + strconv.Itoa(run) + " successfully executed")
	}
	log.Println(LoggerHdr + "Saving tcpdump output for run " + strconv.Itoa(run) + " to file")
	_, err = tcpRtt.Write(tcpdumpOutput)
	if err != nil {
		log.Println(LoggerHdr+"*** ERROR saving tcpdump output for run "+strconv.Itoa(run)+" to file", err)
	} else {
		log.Println(LoggerHdr + "Tcpdump output for run " + strconv.Itoa(run) + " successfully saved to file")
	}
	wg.Done()
}

func getTimestamp() time.Time {
	return time.Now()
}

// Get preferred outbound ip of this machine
func getOutboundIP() string {
	const LoggerHdr = "@getOutboundIP - "

	log.Println(LoggerHdr + "Retrieving default outbound IP")
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(LoggerHdr+"*** ERROR dialing to understand default outbound IP:", err)
	} else {
		log.Println(LoggerHdr + "Outbound IP successfully retrieved")
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
