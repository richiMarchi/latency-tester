FROM golang as builder
RUN apt-get update && apt-get install libpcap-dev -y
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY client/go.mod .
COPY client/go.sum .
RUN go mod download
COPY . .
RUN go build -o script . && cd client && go build -o client .

FROM ubuntu
RUN apt-get update && apt-get install -y traceroute tcpdump iperf3 iputils-ping tshark
COPY --from=builder /build/script .
COPY --from=builder /build/client .
ENTRYPOINT ["./script"]
