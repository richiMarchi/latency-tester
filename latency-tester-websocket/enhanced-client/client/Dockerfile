FROM golang as builder
COPY serialization/ /usr/local/go/src/serialization
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o client .

FROM ubuntu
RUN apt-get update && apt-get install -y traceroute tcpdump iperf3
COPY --from=builder /build/client .
ENTRYPOINT ["./client"]
