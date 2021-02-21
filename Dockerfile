# first image used to build the sources
FROM golang:1.15.5-buster AS builder

ENV GO111MODULE=on \
  GOOS=linux \
  CGO_ENABLED=0 \
  GOARCH=amd64

WORKDIR /tor-proxy

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .


RUN go build -ldflags="-s -w" -o torproxy ./cmd/*

WORKDIR /build


RUN cp /tor-proxy/torproxy .

FROM debian:buster


RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /bin/torproxy /

RUN install /torproxy /bin


ENTRYPOINT ["/torproxy"]

