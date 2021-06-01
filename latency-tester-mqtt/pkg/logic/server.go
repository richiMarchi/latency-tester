package logic

import (
	"crypto/rand"
	"time"

	serialization "github.com/richiMarchi/latency-tester/latency-tester-mqtt/pkg/message/serialization/protobuf"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/klog/v2"
)

type ServerSubscriber struct {
	client  mqtt.Client
	qos     byte
	payload []byte
}

func NewServerSubscriber(client mqtt.Client, responseSize uint, qos byte) *ServerSubscriber {
	payload := make([]byte, responseSize)
	if _, err := rand.Read(payload); err != nil {
		klog.Fatal("Failed to build payload", err)
	}

	return &ServerSubscriber{
		client:  client,
		qos:     qos,
		payload: payload,
	}
}

func (s *ServerSubscriber) Subscribe() {
	klog.Infof("Subscribing to topic: %s", RequestTopic)
	token := s.client.Subscribe(RequestTopic, s.qos, s.onMessage)
	token.Wait()
	klog.Infof("Subscribed to topic: %s", RequestTopic)
}

func (s *ServerSubscriber) onMessage(client mqtt.Client, msg mqtt.Message) {
	request := &serialization.Message{}
	err := proto.Unmarshal(msg.Payload(), request)
	if err != nil {
		klog.Errorf("Failed to unmarshal message: %v", err)
	}

	klog.Infof("Received message with ID %v", request.Id)
	s.publishResponse(request)
}

func (s *ServerSubscriber) publishResponse(response *serialization.Message) {
	response.ServerTimestamp = timestamppb.Now()
	response.Payload = s.payload

	marshal, err := proto.Marshal(response)
	if err != nil {
		klog.Errorf("Failed to marshal message: %v", err)
	}

	token := s.client.Publish(ResponseTopic, s.qos, false, marshal)
	go func(id uint, start time.Time) {
		<-token.Done()
		klog.V(3).Infof("Confirmed publication of message %v in %v", id, time.Since(start))
		if token.Error() != nil {
			klog.Error("Failed to publish message %v: ", id, token.Error())
		}
	}(uint(response.Id), time.Now())
}
