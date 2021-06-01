package logic

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	serialization "github.com/richiMarchi/latency-tester/latency-tester-mqtt/pkg/message/serialization/protobuf"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/klog/v2"
)

type ClientRequester struct {
	client           mqtt.Client
	currentMessageID uint
	repetitions      uint
	intervalMs       uint
	qos              byte
	payload          []byte

	shutdown chan os.Signal
}

type ClientSubscriber struct {
	client      mqtt.Client
	output      *os.File
	repetitions uint
	qos         byte
	received    uint

	shutdown chan os.Signal
}

func NewClientRequester(client mqtt.Client, repetitions, interval, requestSize uint, qos byte, shutdown chan os.Signal) *ClientRequester {
	payload := make([]byte, requestSize)
	if _, err := rand.Read(payload); err != nil {
		klog.Fatal("Failed to build payload", err)
	}

	return &ClientRequester{
		client:           client,
		currentMessageID: 0,
		repetitions:      repetitions,
		intervalMs:       interval,
		payload:          payload,
		qos:              qos,
		shutdown:         shutdown,
	}
}

func NewClientSubscriber(client mqtt.Client, outputFile string, repetitions uint, qos byte, shutdown chan os.Signal) *ClientSubscriber {
	file, err := os.Create(outputFile)
	if err != nil {
		klog.Fatal("Failed to open output file: ", err)
	}
	_, _ = file.WriteString("client-send-timestamp,server-timestamp,e2e-rtt\n")

	return &ClientSubscriber{
		client:      client,
		output:      file,
		repetitions: repetitions,
		received:    0,
		qos:         qos,
		shutdown:    shutdown,
	}
}

func (r *ClientRequester) PublishRequests() {
	klog.Infof("Starting to publish requests on %s", RequestTopic)
	for r.currentMessageID = 0; r.currentMessageID < r.repetitions; r.currentMessageID++ {
		start := time.Now()
		r.publishRequest(&start)

		waitTime := time.Millisecond*time.Duration(r.intervalMs) - time.Since(start)
		if waitTime < 0 {
			klog.Warningf("Missed deadline when sending message %v", r.currentMessageID)
			waitTime = 0
		}

		if r.currentMessageID == r.repetitions-1 {
			break
		}

		select {
		case signal := <-r.shutdown:
			klog.Info("Signal caught - exiting")
			r.shutdown <- signal
			return
		case <-time.After(waitTime):
		}
	}
	klog.Infof("Finished to publish requests on %s", RequestTopic)
}

func (r *ClientRequester) publishRequest(now *time.Time) {
	message := &serialization.Message{
		Id:              int32(r.currentMessageID),
		ClientTimestamp: timestamppb.New(*now),
		ServerTimestamp: &timestamppb.Timestamp{},
		Payload:         r.payload,
	}

	marshal, err := proto.Marshal(message)
	if err != nil {
		klog.Errorf("Failed to marshal message: %v", err)
	}
	klog.V(3).Infof("Publishing message %v", r.currentMessageID)
	token := r.client.Publish(RequestTopic, r.qos, false, marshal)
	go func(id uint) {
		<-token.Done()
		klog.V(3).Infof("Confirmed publication of message %v in %v", id, time.Since(*now))
		if token.Error() != nil {
			klog.Error("Failed to publish message %v: ", id, token.Error())
		}
	}(r.currentMessageID)
}

func (s *ClientSubscriber) Subscribe() {
	klog.Infof("Subscribing to topic: %s", ResponseTopic)
	token := s.client.Subscribe(ResponseTopic, s.qos, s.onMessage)
	token.Wait()
	klog.Infof("Subscribed to topic: %s", ResponseTopic)
}

func (s *ClientSubscriber) Cleanup() {
	s.output.Close()
}

func (s *ClientSubscriber) onMessage(client mqtt.Client, msg mqtt.Message) {
	response := &serialization.Message{}
	err := proto.Unmarshal(msg.Payload(), response)
	if err != nil {
		klog.Errorf("Failed to unmarshal message: %v", err)
	}

	latency := time.Since(response.ClientTimestamp.AsTime())
	latencyMs := float64(latency.Nanoseconds()) / float64(time.Millisecond.Nanoseconds())

	klog.Infof("Received message %d in %.2f ms", response.Id, latencyMs)

	outputStr := fmt.Sprintf("%v,%v,%f\n",
		response.ClientTimestamp.AsTime().UnixNano(),
		response.ServerTimestamp.AsTime().UnixNano(),
		latencyMs)
	_, _ = s.output.WriteString(outputStr)

	s.received += 1
	if s.received == s.repetitions {
		klog.Infof("All responses received")
		s.shutdown <- os.Interrupt
	}
}
