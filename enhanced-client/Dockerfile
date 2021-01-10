FROM golang as builder
RUN apt-get update && apt-get install libpcap-dev -y
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY client/go.mod .
COPY client/go.sum .
RUN go mod download
COPY plotter/go.mod .
COPY plotter/go.sum .
RUN go mod download
COPY . .
RUN go build -o enhanced-client . && cd client && go build -o client . && cd ../plotter && go build -o plotter .

FROM ubuntu
RUN apt-get update && apt-get install -y traceroute tcpdump iperf3 iputils-ping tshark
COPY --from=builder /build/enhanced-client .
COPY --from=builder /build/client .
COPY --from=builder /build/plotter .
ENTRYPOINT ["./enhanced-client"]
