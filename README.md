# Latency Tester

GoLang tool designed to test the latency between a client and a server.
The two endpoints interact using Websocket as communication protocol: it provides a full-duplex communication channel over a single TCP connection.
The tool can run with different parameters set as command line arguments.

Collected metrics (.csv file as output):
* E2E application delay with Websocket
* OS RTT with ping
* TCP main socket parameters

If requested, the result of traceroute can be retrieved too.

## How to run the docker

### Server

```
docker pull richimarchi/latency-tester_server
docker run -p 8080:8080 [--name <container-name>] richimarchi/latency-tester_server:latest [-addr=<ip:port>] [-tls=<enabled>]
```

#### Server flags:

|Param|Description|Default Value|
|---|---|---|
|`-addr`|Listening address and port|`0.0.0.0:8080`|
|`-tls`|`true` if TLS requested|`false`|

### Client

```
docker pull richimarchi/latency-tester_client
docker run -p 8080:8080 [--name <container-name>] -v <local-log-folder>:/tmp richimarchi/latency-tester_client:latest [-reps=<repetitions>] [-requestPayload=<bytes>] [-responsePayload=<bytes>] [-interval=<ms>] [-tcpStats=<enabled>] [-tls=<enabled>] [-traceroute=<enabled>] [-log=<log-file>] <address> <ping-ip>
```

#### Required input parameters

|Param|Description|
|---|---|
|`<address>`|Address of the running server|
|`<ping-ip>`|Address of the node to ping or traceroute|

#### Client flags:

|Param|Description|Default Value|
|---|---|---|
|`-reps`|Number of test repetition, if `0` it runs until given interrupt (`CTRL + C`)|`0`|
|`-requestPayload`|Request payload size (in bytes), minimum value: `62`|`64`|
|`-responsePayload`|Response payload size (in bytes), minimum value: `62`|`64`|
|`-interval`|Requests send interval (in milliseconds)|`1000`|
|`-tcpStats`|`true` if TCP Stats requested (short execution time is suggested, as it consumes a lot of CPU and RAM)|`false`|
|`-tls`|`true` if TLS requested|`false`|
|`-traceroute`|`true` if traceroute is requested to run|`false`|
|`-log`|Define the name of the file|`log`|

## How to deploy the server into a Kubernetes cluster

Starting from the `serverDeploymentSkeleton.yaml` file, generate the custom deployment file depending on your hostname and deploy it in your k8s cluster:

```
export HOSTNAME=<custom-hostname>
envsubst < serverDeploymentSkeleton.yaml > customServerDeployment.yaml
kubectl apply -f customServerDeployment.yaml
```

*N.B.: Ingress annotations are ingress controller dependent*
