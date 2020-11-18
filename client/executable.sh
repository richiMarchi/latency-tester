#!/bin/bash

if [ $# -eq 0 ]; then
  echo "No arguments supplied: how many hours should the tool run?"
  exit 1
fi

if ! [[ "$1" =~ ^[0-9]+$ ]] || (( $1 < 1)); then
  echo "Sorry, positive integers only"
  exit 1
fi

# Test the bottleneck bandwidth
echo "Executing iperf to test bottleneck bandwidth..."
iperf3 -c 130.192.31.240 -p 80 > /tmp/iperf3_report.txt
echo "iperf complete!"

# Start ping in background. Remember to stop it before returning!
ping 130.192.31.254 -i 30 -D > /tmp/ping_report.txt &
PING_PID=$!

# Start tcpdump in background. Remember to stop it before returning!
tcpdump -U -s 96 -w /tmp/tcpdump_report.pcap 'net 130.192.31.241/32 and tcp port 443' &
TCPDUMP_PID=$!

# E2E latency testing
endpoints=("192.168.31.102:30011" "192.168.31.104:30011" "130.192.31.242:8080" "latency-tester.crownlabs.polito.it")
intervals=(10 25 50 100 250 500)
sizes=(1024 10240 102400 1024000)

for ((n=1;n<=$1;n++)); do
  start_time="$(date -u +%s)"
  for address in "${endpoints[@]}"; do
    for interval in "${intervals[@]}"; do
      for request in "${sizes[@]}"; do
        repetitions=$((30000/$interval))
        go run *.go -reps=$repetitions -interval=$interval -requestPayload=$request -responsePayload=1024 -tls=true -log="$n-$address.i$interval.x$request" $address
      done
    done
  done
  if [ "$n" -ne "$1" ]; then
    end_time="$(date -u +%s)"
    elapsed="$(($end_time-$start_time))"
    sleep $((3600 - $elapsed))
  fi
done

# Stop background ping and tcpdump
kill -INT $PING_PID
kill -INT $TCPDUMP_PID
