FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build op-node with local monorepo go modules
COPY ./op-node/docker.go.work /app/go.work
COPY ./op-bindings /app/op-bindings
COPY ./op-node /app/op-node
COPY ./op-chain-ops /app/op-chain-ops
COPY ./op-service /app/op-service

WORKDIR /app/op-node

RUN make op-node

FROM alpine:3.15

COPY --from=builder /app/op-node/bin/op-node /usr/local/bin

CMD ["op-node"]
