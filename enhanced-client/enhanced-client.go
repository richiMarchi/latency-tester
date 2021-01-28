package main

import (
	"context"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
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
	TlsEnabled        string         `yaml:"tls_enabled"`
	ExecDir           string         `yaml:"exec_dir"`
}

func main() {

	if len(os.Args) == 1 {
		log.Fatal("Settings filename requested")
	}

	stopHealthChecker := make(chan os.Signal, 1)
	go runHealthChecker(stopHealthChecker)

	// Settings parsing
	file, err := ioutil.ReadFile(os.Args[1])
	errMgmt(err)
	var settings Settings
	err = yaml.Unmarshal(file, &settings)
	errMgmt(err)

	// Print flags statuses in order to be sure it is as expected
	//val, err := sysctl.Get("net.ipv4.tcp_slow_start_after_idle")
	//errMgmt(err)
	//log.Println("TCP slow start after idle value: ", val)
	//val, err = sysctl.Get("net.ipv4.tcp_congestion_control")
	//errMgmt(err)
	//log.Println("TCP congestion control algorithm: ", val)

	// Start ping and tcpdump in background
	stopPing := make(chan os.Signal, len(settings.PingDestinations))
	var wg sync.WaitGroup
	wg.Add(len(settings.PingDestinations))
	killPing := false
	for _, dest := range settings.PingDestinations {
		go pingThread(&wg, settings.ExecDir, dest, settings.PingInterval, stopPing, &killPing)
	}

	for i := 1; i <= settings.Runs; i++ {
		for _, iperfData := range settings.IperfDestinations {
			if iperfData.Port == "" {
				iperfData.Port = "5201"
			}
			log.Println("Running Iperf towards", iperfData.Name+"...")
			iperfer(i, settings.ExecDir, iperfData)
			log.Println("Iperf towards", iperfData.Name, "complete!")
		}
		stopTcpdump := make(chan os.Signal, 1)
		wg.Add(1)
		localIp := getOutboundIP()
		go tcpDumper(i, &wg, stopTcpdump, localIp, settings.ExecDir)
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
					if err != nil {
						log.Print("ERROR: ")
						log.Println(err)
					}
				}
			}
		}
		stopTcpdump <- os.Interrupt
		for range settings.PingDestinations {
			stopPing <- os.Interrupt
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

	killPing = true
	for range settings.PingDestinations {
		stopPing <- os.Interrupt
	}
	wg.Wait()

	// Plotting
	log.Println("Plotting...")
	err = exec.Command("./plotter", os.Args[1]).Run()
	errMgmt(err)
	log.Println("Everything's complete!")

	stopHealthChecker <- os.Interrupt
}

func runHealthChecker(c chan os.Signal) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { return })
	server := http.Server{Addr: "0.0.0.0:8080"}

	// Handle stop
	go func() {
		for range c {
			_ = server.Shutdown(context.TODO())
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
	errMgmt(err)
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
				_ = pingerCmd.Process.Signal(os.Interrupt)
			}
		}()

		pingOutput, err := pingerCmd.Output()
		errMgmt(err)
		_, err = osRtt.Write(pingOutput)
		errMgmt(err)
	}
	wg.Done()
}

func tcpDumper(run int, wg *sync.WaitGroup, c chan os.Signal, localIp, execdir string) {
	tcpRtt, err := os.Create(execdir + strconv.Itoa(run) + "-tcpdump_report.csv")
	errMgmt(err)
	defer tcpRtt.Close()
	tcpRtt.WriteString("#frame-timestamp,tcp-ack-rtt,tcp-stream-id\n")
	tcpdumpCmd := exec.Command("tshark",
		"-ni", "any",
		"-Y", "tcp.analysis.ack_rtt and ip.dst=="+localIp,
		"-e", "frame.time_epoch",
		"-e", "tcp.analysis.ack_rtt",
		"-e", "tcp.stream",
		"-T", "fields",
		"-E", "separator=,",
		"-E", "quote=d")

	// Handle stop
	go func() {
		for range c {
			_ = tcpdumpCmd.Process.Signal(os.Interrupt)
		}
	}()

	tcpdumpOutput, err := tcpdumpCmd.Output()
	errMgmt(err)
	_, err = tcpRtt.Write(tcpdumpOutput)
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
