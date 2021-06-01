package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/richiMarchi/latency-tester/latency-tester-mqtt/pkg/logic"
	"k8s.io/klog/v2"
)

const id = "latency-tester-client"

func main() {
	broker := flag.String("broker", "", "The address to contact the broker")
	username := flag.String("username", "", "The broker username")
	password := flag.String("password", "", "The broker password")
	repetitions := flag.Uint("reps", 1, "number of repetitions")
	interval := flag.Uint("interval", 100, "send interval time (ms)")
	requestSize := flag.Uint("requestSize", 1024, "bytes of the payload")
	log := flag.String("log", "./log.csv", "file to store latency results")
	qos := flag.Uint("qos", 0, "mqtt QoS")
	klog.InitFlags(nil)
	flag.Parse()

	klog.Infof("Broker: %v", *broker)
	klog.Infof("Repetitions: %v", *repetitions)
	klog.Infof("Interval: %v ms", *interval)
	klog.Infof("Request Size: %v Bytes", *requestSize)
	klog.Infof("QoS: %v", byte(*qos))

	logic.ConfigureLogging()

	opts := logic.BuildCommonConnectionOptions(*broker, id, *username, *password)
	client, err := logic.EstablishBrokerConnection(opts)
	if err != nil {
		klog.Fatal(err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	signal.Notify(shutdown, syscall.SIGTERM)

	subscriber := logic.NewClientSubscriber(client, *log, *repetitions, byte(*qos), shutdown)
	subscriber.Subscribe()

	requester := logic.NewClientRequester(client, *repetitions, *interval, *requestSize, byte(*qos), shutdown)
	requester.PublishRequests()

	<-shutdown
	klog.Info("Exiting")
	client.Disconnect(logic.DisconnectQuiescence)
}
