FROM golang as builder
RUN apt-get update && apt-get install libpcap-dev -y
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o plotter .

FROM ubuntu
COPY --from=builder /build/plotter .
ENTRYPOINT ["./plotter"]
