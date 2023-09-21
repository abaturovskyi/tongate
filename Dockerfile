FROM docker.io/library/golang:1.19.2 AS builder
WORKDIR /build-dir
COPY go.mod .
COPY go.sum .
RUN go get ./...
COPY main.go main.go
COPY api api
COPY internal/config internal/config
RUN go build -o /tmp/tongate .

FROM alpine:latest AS tongate
# RUN apt-get update && apt-get -y install zlib1g-dev libssl-dev openssl ca-certificates && rm -rf /var/lib/apt/lists/*
# RUN mkdir -p /lib
COPY --from=builder /tmp/tongate /app/tongate
# ENV LD_LIBRARY_PATH=/lib
CMD ["/app/tongate", "-v"]
