FROM golang as builder

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o client cmd/client/client.go

FROM alpine:3
COPY --from=builder /build/client /
ENTRYPOINT ["/client"]
