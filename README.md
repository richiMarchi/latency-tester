# Latency Tester

GoLang tool designed to test the latency between a client and a server.
The two endpoints interact using Websocket as communication protocol: it provides a full-duplex communication channel over a single TCP connection.
The tool can run with different parameters set as command line arguments.

Collected metrics (.csv file as output):
* E2E application delay with Websocket
* TCP main socket parameters

The client is wrapped around by an enhanced version of it, that is able to run the client in the most powerful way by
defining many parameters in a single YAML file and generates few helpful plots to analyse data.

The structure of the project repository is:

- [Enhanced Client](enhanced-client)
  - [Client](enhanced-client/client)
  - [Plotter](enhanced-client/plotter)
- [Server](server)

What is this tool capable of? [Look at these examples](enhanced-client/plotter#plotter-output-examples)
