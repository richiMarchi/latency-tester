package logic

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"
)

const (
	RequestTopic  = "latency-tester/request"
	ResponseTopic = "latency-tester/response"

	DisconnectQuiescence = 100 // ms
)

type DebugLogger struct{}
type WarningLogger struct{}
type ErrorLogger struct{}
type FatalLogger struct{}

func (*DebugLogger) Println(v ...interface{})                 { klog.V(5).Info(v...) }
func (*DebugLogger) Printf(format string, v ...interface{})   { klog.V(5).Infof(format, v...) }
func (*WarningLogger) Println(v ...interface{})               { klog.V(2).Info(v...) }
func (*WarningLogger) Printf(format string, v ...interface{}) { klog.V(2).Infof(format, v...) }
func (*ErrorLogger) Println(v ...interface{})                 { klog.V(1).Info(v...) }
func (*ErrorLogger) Printf(format string, v ...interface{})   { klog.V(1).Infof(format, v...) }
func (*FatalLogger) Println(v ...interface{})                 { klog.Error(v...) }
func (*FatalLogger) Printf(format string, v ...interface{})   { klog.Errorf(format, v...) }

func ConfigureLogging() {
	mqtt.DEBUG = &DebugLogger{}
	mqtt.WARN = &WarningLogger{}
	mqtt.ERROR = &ErrorLogger{}
	mqtt.CRITICAL = &FatalLogger{}
}

func BuildCommonConnectionOptions(broker, id, username, password string) *mqtt.ClientOptions {
	return mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(id).
		SetUsername(username).
		SetPassword(password).
		SetDefaultPublishHandler(defaultMessageHandler).
		SetOnConnectHandler(defaultConnectionHandler).
		SetConnectionLostHandler(defaultLostHandler).
		SetCleanSession(true).
		SetOrderMatters(false)
}

func EstablishBrokerConnection(opts *mqtt.ClientOptions) (mqtt.Client, error) {
	klog.Info("Establishing broker connection")
	client := mqtt.NewClient(opts)
	token := client.Connect()

	<-token.Done()
	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}

func defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	klog.Warningf("Received message: %s from unmanaged topic: %s", msg.Payload(), msg.Topic())
}

func defaultConnectionHandler(client mqtt.Client) {
	klog.Info("Broker connection correctly established")
}

func defaultLostHandler(client mqtt.Client, err error) {
	klog.Warningf("Connect lost: %v", err)
}
