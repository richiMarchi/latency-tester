FROM golang as builder
COPY serialization/ /usr/local/go/src/serialization
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o server .
RUN apt-get update && apt-get install openssl -y && openssl genrsa -out server.key 2048 \
    && openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -subj "/C=/ST=/L=/O=/OU=/CN="

FROM ubuntu
COPY --from=builder /build/server.crt .
COPY --from=builder /build/server.key .
COPY --from=builder /build/server .
EXPOSE 8080
ENTRYPOINT ["./server"]
