FROM golang as builder
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o server .

FROM ubuntu
RUN apt-get update && apt-get install openssl -y && openssl genrsa -out server.key 2048 \
    && openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -subj "/C=/ST=/L=/O=/OU=/CN="
COPY --from=builder /build/server .
EXPOSE 8080
ENTRYPOINT ["./server"]
