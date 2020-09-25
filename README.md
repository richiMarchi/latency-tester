# Latency Tester
WebSocket based latency tester

## How to run the docker

Server

```
docker pull richimarchi/latency-tester_server
docker run -p 8080:8080 --rm richimarchi/latency-tester_server:latest [-addr=<ip:port>]
```

Client

```
docker pull richimarchi/latency-tester_client
docker run -p 8080:8080 -v <local-log-folder>:/log --rm richimarchi/latency-tester_client:latest [-reps=<repetitions>] [-sendPayload=<bytes>] [-responsePayload=<bytes>] [-interval=<ms>] [-log=<log-file>] <ip:port>
```
