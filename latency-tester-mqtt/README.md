# Latency Tester MQTT

A GoLang tool designed to measure the latency between a client and a server using the MQTT publish/subscribe abstraction
for the communication. It leverages a ping-pong approach, with a client publishing to a first topic and measuring the delay
before the response is echoed back by the server on a second one. The results are written to a csv file for further analysis.

## Degrees of configuration

Both the client and the server components can be configured by means of command-line flags.
Specifically, the main parameters are the following:

* The MQTT broker address, and optionally its credentials;
* The size of the request and response messages;
* The delay between two subsequent messages published by the client;
* The number of messages published by the client;
* The selected MQTT QoS level.

## How to build the tool

Both client and server can be built using the go compiler:

```bash
CGO_ENABLED=0 go build -o latency-tester-mqtt-client cmd/client/client.go
CGO_ENABLED=0 go build -o latency-tester-mqtt-server cmd/server/server.go
```

Alternatively, it is possible to build the corresponding docker images:

```bash
docker build -t latency-tester-mqtt-client -f build/client/Dockerfile .
docker build -t latency-tester-mqtt-server -f build/server/Dockerfile .
```
