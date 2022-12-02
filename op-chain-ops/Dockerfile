FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./op-chain-ops/docker.go.work /app/go.work
COPY ./op-chain-ops /app/op-chain-ops
COPY ./op-bindings /app/op-bindings
COPY ./op-node /app/op-node

WORKDIR /app/op-chain-ops

RUN make op-migrate

FROM alpine:3.15

COPY --from=builder /app/op-chain-ops/bin/op-migrate /usr/local/bin

ENTRYPOINT ["op-migrate"]
