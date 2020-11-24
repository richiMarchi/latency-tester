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
RUN go build -o client . && cd plotter && go build -o plotter .


# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/client . && cp /build/executable.sh . && cp /build/plotter/plotter .

# Run the executable
ENTRYPOINT ["./executable.sh"]