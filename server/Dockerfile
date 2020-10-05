FROM golang

RUN apt-get update && openssl genrsa -out server.key 2048 \
    && openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -subj "/C=/ST=/L=/O=/OU=/CN="

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Download all the dependencies
RUN go get -d -v ./...

# Build the Go app
RUN go build -o server .

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the executable
ENTRYPOINT ["./server"]