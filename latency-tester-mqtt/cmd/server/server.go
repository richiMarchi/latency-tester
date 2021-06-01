package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/richiMarchi/latency-tester/latency-tester-mqtt/pkg/logic"
	"k8s.io/klog/v2"
)

const id = "latency-tester-server"

func main() {
	broker := flag.String("broker", "", "The address to contact the broker")
	username := flag.String("username", "", "The broker username")
	password := flag.String("password", "", "The broker password")
	responseSize := flag.Uint("responseSize", 1024, "bytes of the payload")
	qos := flag.Uint("qos", 0, "mqtt QoS")
	klog.InitFlags(nil)
	flag.Parse()

	klog.Infof("Broker: %v", *broker)
	klog.Infof("Response Size: %v Bytes", *responseSize)
	klog.Infof("QoS: %v", byte(*qos))

	logic.ConfigureLogging()

	opts := logic.BuildCommonConnectionOptions(*broker, id, *username, *password)
	client, err := logic.EstablishBrokerConnection(opts)
	if err != nil {
		klog.Fatal(err)
	}

	subscriber := logic.NewServerSubscriber(client, *responseSize, byte(*qos))
	subscriber.Subscribe()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	signal.Notify(shutdown, syscall.SIGTERM)

	<-shutdown
	klog.Info("Signal caught - exiting")
	client.Disconnect(logic.DisconnectQuiescence)
}
