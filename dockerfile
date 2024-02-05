############################################################
# Dockerfile to build golang Installed Containers

# Based on alpine

############################################################

FROM golang:1.21 AS builder

COPY . /src
WORKDIR /src

RUN GOPROXY="https://goproxy.cn,direct" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine:3.13

RUN mkdir /sidecar
COPY --from=builder /src/sidecar /sidecar

EXPOSE 5044
WORKDIR /sidecar
CMD ["/sidecar watch"]
