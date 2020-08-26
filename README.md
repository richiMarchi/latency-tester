# Latency Tester
WebSocket based latency tester

##How to run the docker (locally)
#####NB: at the moment args management is missing

Server

`docker run -p 8080:8080 --rm richimarchi/latency-tester_server:latest [-addr=<ip:port>] [-interval=<ms>]`

Client

`docker run -p 8080:8080 -v /home/rick/lat-test:/log --rm richimarchi/latency-tester_client:latest [-reps=<repetitions>] 
[-payload=<bytes>] [-interval=<ms>] [-log=<log-file>] <ip:port>`
