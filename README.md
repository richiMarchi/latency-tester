# Latency Tester

GoLang tool designed to test the latency between a client and a server.
The two endpoints interact using Websocket as communication protocol: it provides a full-duplex communication channel 
over a single TCP connection, therefore avoiding creating a new one for each message.
The tool can run with different parameters set as command line arguments.

Collected metrics:
* E2E application delay with Websocket
* TCP delay with Tshark
* Network delay with Ping
* Network Bandwidth with Iperf3

The client is wrapped around by an enhanced version of it, that is able to run the client in the most powerful way by
defining many parameters in a single YAML file and generates few helpful plots to analyse data.

The deployment of both enhanced client and server can be done with the source code, but it is easier using the docker 
images provided on the public Docker Hub, so that it is also possible to deploy the components inside a Kubernetes
cluster. To help you do that, each component has its own deployment documentation in the corresponding directory.

The structure of the project repository is:

- [Enhanced Client](enhanced-client)
  - [Client](enhanced-client/client)
  - [Plotter](enhanced-client/plotter)
- [Server](server)

What is this tool capable of? [Look at these examples](enhanced-client/plotter#plotter-output-examples)
