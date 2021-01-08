package main

import (
	"bytes"
	"encoding/csv"
	"github.com/lorenzosaino/go-sysctl"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgpdf"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
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
	ExecDir      string `yaml:"exec_dir"`
}

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

	// Print flags statuses in order to be sure it is as expected
	val, err := sysctl.Get("net.ipv4.tcp_slow_start_after_idle")
	errMgmt(err)
	log.Println("TCP slow start after idle value: ", val)
	val, err = sysctl.Get("net.ipv4.tcp_congestion_control")
	errMgmt(err)
	log.Println("TCP congestion control algorithm: ", val)

	// Start ping and tcpdump in background
	stopPing := make(chan os.Signal, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go pingThread(&wg, settings.ExecDir, settings.PingIp, settings.PingInterval, stopPing)

	for i := 1; i <= settings.Runs; i++ {
		log.Println("Running Iperf...")
		iperfer(i, settings.ExecDir, settings.IperfIp, settings.IperfPort)
		log.Println("Iperf complete!")
		stopTcpdump := make(chan os.Signal, 1)
		wg.Add(1)
		go tcpDumper(i, settings, &wg, stopTcpdump, settings.ExecDir)
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
		stopTcpdump <- os.Interrupt
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

	stopPing <- os.Interrupt
	wg.Wait()

	// Plotting
	log.Println("Plotting...")
	err = exec.Command("./plotter", os.Args[1]).Run()
	errMgmt(err)
	log.Println("Everything's complete!")
}

func iperfer(run int, execdir, ip, port string) {
	iperfFile, err := os.Create(execdir + strconv.Itoa(run) + "-iperf_report.txt")
	errMgmt(err)
	defer iperfFile.Close()

	iperfRes, err := exec.Command("timeout", "10", "iperf3", "-c", ip, "-p", port, "-t", "5").Output()
	_, err = iperfFile.Write(iperfRes)
	errMgmt(err)
}

func pingThread(wg *sync.WaitGroup, execdir, address string, interval int, c chan os.Signal) {
	osRtt, err := os.Create(execdir + "ping_report.txt")
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

func tcpDumper(run int, settings Settings, wg *sync.WaitGroup, c chan os.Signal, execdir string) {
	tcpdumper := exec.Command("tshark",
		"-ni", "any",
		"-Y", "tcp.analysis.ack_rtt and ip.dst==172.0.0.0/8",
		"-e", "frame.time_epoch",
		"-e", "tcp.analysis.ack_rtt",
		"-e", "tcp.stream",
		"-T", "fields",
		"-E", "separator=,",
		"-E", "quote=d")
	var out bytes.Buffer
	var stderr bytes.Buffer
	tcpdumper.Stdout = &out
	tcpdumper.Stderr = &stderr

	// Handle stop
	go func() {
		for range c {
			_ = tcpdumper.Process.Signal(os.Interrupt)
		}
	}()

	err := tcpdumper.Run()
	if err != nil {
		log.Println(stderr.String())
	}
	errMgmt(err)

	var values plotter.XYs
	var firstTs float64
	var previousStream int
	streamCounter := 0
	records, _ := csv.NewReader(bytes.NewReader(out.Bytes())).ReadAll()

	pdfToSave := vgpdf.New(vg.Points(2000), vg.Points(1000))
	w, err := os.Create(execdir + strconv.Itoa(run) + "-tcpPlot.pdf")
	if err != nil {
		panic(err)
	}

	for index, row := range records {
		ts, fail := strconv.ParseFloat(row[0], 64)
		if fail != nil {
			continue
		}
		rtt, fail := strconv.ParseFloat(row[1], 64)
		if fail != nil {
			continue
		}
		streamId, _ := strconv.Atoi(row[2])
		if len(values) == 0 {
			firstTs = ts
			previousStream = streamId
		}
		if previousStream != streamId || index == len(records)-1 {
			if streamCounter != 0 {
				pdfToSave.NextPage()
			}
			// If it is the last iteration, add the last record before saving to pdf
			if index == len(records)-1 {
				values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
			}
			p, err := plot.New()
			errMgmt(err)
			p.X.Label.Text = "Time (s)"
			p.Y.Label.Text = "TCP RTT (ms)"
			p.Y.Tick.Marker = hplot.Ticks{N: 15}
			p.X.Tick.Marker = hplot.Ticks{N: 15}
			tracker := 0
			for _, addr := range settings.Endpoints {
				for _, inter := range settings.Intervals {
					for _, size := range settings.MsgSizes {
						if tracker == streamCounter {
							p.Title.Text = "TCP ACK Latency: " + addr.Description + " - " + strconv.Itoa(inter) + "ms - " + strconv.Itoa(size) + "B"
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
			sort.Slice(values, func(i, j int) bool {
				return values[i].Y < values[j].Y
			})
			toRemove := len(values) / 100
			values = values[:len(values)-toRemove*3]
			sort.Slice(values, func(i, j int) bool {
				return values[i].X < values[j].X
			})
			err = plotutil.AddLines(p, "ACK RTT", values)
			p.Draw(draw.New(pdfToSave))
			values = values[:0]
			streamCounter += 1
			previousStream = streamId
		}
		values = append(values, plotter.XY{X: ts - firstTs, Y: rtt * 1000})
	}

	if _, err := pdfToSave.WriteTo(w); err != nil {
		panic(err)
	}
	w.Close()
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
