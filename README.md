# Latency Tester
WebSocket based latency tester

## How to run the docker

### Server

```
docker pull richimarchi/latency-tester_server
docker run -p 8080:8080 [--name <container-name>] richimarchi/latency-tester_server:latest [-addr=<ip:port>] [-tls=<enabled>]
```

#### Default input parameters:

Listening address and port:
`-addr` = `0.0.0.0:8080`

TLS enabled:
`-tls` = `false`

### Client

```
docker pull richimarchi/latency-tester_client
docker run -p 8080:8080 [--name <container-name>] -v <local-log-folder>:/tmp richimarchi/latency-tester_client:latest [-reps=<repetitions>] [-requestPayload=<bytes>] [-responsePayload=<bytes>] [-interval=<ms>] [-tls=<enabled>] [-traceroute=<enabled>] [-log=<log-file>] <ip:port> <ping/traceroute-ip>
```

#### Default input parameters:

Number of test repetitions:
`-reps` = `0` (infinite iterations)

Request payload size (in bytes):
`-requestPayload` = `64`

Response payload size (in bytes):
`-responsePayload` = `64`

Requests send interval (in milliseconds):
`-interval` = `1000`

TLS enabled:
`-tls` = `false`

Traceroute enabled:
`-traceroute` = `false`

Log file name:
`-log` = `log`


## How to deploy the server into a Kubernetes cluster

Starting from the `serverDeploymentSkeleton.yaml` file, generate the custom deployment file depending on your hostname and deploy it in your k8s cluster:

```
export HOSTNAME=<custom-hostname>
envsubst '$HOSTNAME' < serverDeploymentSkeleton.yaml > customServerDeployment.yaml
kubectl apply -f customServerDeployment.yaml
```

*N.B.: Ingress annotations are ingress controller dependent*
