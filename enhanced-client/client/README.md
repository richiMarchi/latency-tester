# Latency Tester Client

The client is a complex but easy to run executable, flexible with its parameters and efficient for the language used and
the protocol chosen. It is useful in cloud environments, where we are able to consistently calculate the latency towards
a pod instead of having to rely on the standard ping tool, which usually cannot even be used.

Initially it sets server response packet size with a control message. Then it starts sending the packets requested by
the user input flags and stores the RTT inside a csv file. If requested it stores TCP socket statistics and a traceroute
output too. **Beware of the TCP stats, because if the execution is long, the size of the output will be incredibly
huge!**

## How to deploy

```
docker pull richimarchi/latency-tester_client
docker run [--name <container-name>] -v <local-log-folder>:/execdir richimarchi/latency-tester_client [-reps=<repetitions>] [-requestPayload=<bytes>] [-responsePayload=<bytes>] [-interval=<ms>] [-tcpStats=<enabled>] [-tls=<enabled>] [-traceroute=<address>] [-log=<log-file>] <address>
```

Latest version: `1.0.2`

### Required input parameters

|Param|Description|
|---|---|
|`<address>`|Address of the running server|

### Client flags:

|Param|Description|Default Value|
|---|---|---|
|`-reps`|Number of test repetition, if `0` it runs until given interrupt (`CTRL + C`)|`0`|
|`-requestPayload`|Request payload size (in bytes), minimum value: `62`|`64`|
|`-responsePayload`|Response payload size (in bytes), minimum value: `62`|`64`|
|`-interval`|Requests send interval (in milliseconds)|`1000`|
|`-tcpStats`|`true` if TCP Stats requested (short execution time is suggested, as it consumes a lot of CPU and RAM)|`false`|
|`-tls`|`true` if TLS requested|`false`|
|`-traceroute`|If present, address traceroute should run towards||
|`-log`|Define the name of the file|`log`|
