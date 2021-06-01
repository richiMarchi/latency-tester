# Latency Tester

This repository contains two different measurement tools to evaluate the latency between a given pair of endpoints:

* [latency-tester-websocket](./latency-tester-websocket): it measures the latency at different levels of the stack, including
  network (by means of the `ping` utility), TCP (through the analysis capabilities of `tshark`) and application. It is composed
  of a client and a server component, which interact through a persistent, WebSocket-based, communication channel, with message
  sizes and frequency configurable by the user.
* [latency-tester-mqtt](./latency-tester-mqtt): it measures the application level latency experienced by a client and a server
  communicating through MQTT publish/subscribe abstraction. Message sizes, QoS and frequency are configurable by the user.

For more information about the measurement tools, as well as for the build instructions, please refer to the README file in the corresponding folder.

These measurement tools have been used to collect the data presented in the manuscript "When Latency Matters: Measurements
and Lessons Learned" submitted to ACM SIGCOMM Computer Communications Review. The full datasets, along with the post-processing
scripts, are available in a [separate GitHub repository](https://github.com/netgroup-polito/when-latency-matters-datasets),
given their relevant size (>2GB)
