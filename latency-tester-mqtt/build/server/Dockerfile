FROM golang as builder

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o server cmd/server/server.go

FROM alpine:3
COPY --from=builder /build/server /
ENTRYPOINT ["/server"]
