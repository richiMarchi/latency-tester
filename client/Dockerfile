FROM golang

RUN apt-get update && apt-get install -y traceroute tcpdump iperf3

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Download all the dependencies
RUN go get -d -v ./...

# Run the executable
ENTRYPOINT ["./executable.sh"]