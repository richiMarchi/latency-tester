FROM golang

RUN apt-get update && apt-get install -y traceroute tcpdump iperf3

WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o client .

# Run the executable
ENTRYPOINT ["./client"]