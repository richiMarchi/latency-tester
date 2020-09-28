# Latency Tester
WebSocket based latency tester

## How to run the docker

### Server

```
docker pull richimarchi/latency-tester_server
docker run -p 8080:8080 [--name <container-name>] richimarchi/latency-tester_server:latest [-addr=<ip:port>]
```

#### Default input parameters:

Listening address and port:
`addr` = `0.0.0.0:8080`

### Client

```
docker pull richimarchi/latency-tester_client
docker run -p 8080:8080 [--name <container-name>] -v <local-log-folder>:/tmp richimarchi/latency-tester_client:latest [-reps=<repetitions>] [-requestPayload=<bytes>] [-responsePayload=<bytes>] [-interval=<ms>] [-log=<log-file>] <ip:port>
```

#### Default input parameters:

Number of test repetitions:
`-reps` = `0` (infinite iterations)

Request payload size:
`-requestPayload` = `64`

Response payload size:
`-responsePayload` = `64`

Requests send interval:
`-interval` = `1000`

Log file name:
`-log` = `log.csv`
