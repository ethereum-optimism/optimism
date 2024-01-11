# syntax = docker/dockerfile:1.2
FROM docker.io/library/golang:1.20-alpine3.17 AS builder

RUN apk --no-cache add build-base linux-headers git bash ca-certificates libstdc++

ADD ./boba-chain-ops /app/boba-chain-ops
COPY ./boba-bindings /app/boba-bindings
RUN cd /app/boba-chain-ops && make build

FROM alpine:3.13

RUN apk add --no-cache ca-certificates jq curl libgcc libstdc++
COPY --from=builder /app/boba-chain-ops/bin /usr/local/bin/

RUN mkdir /jwt
RUN mkdir /addr
