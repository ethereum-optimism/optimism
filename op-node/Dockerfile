FROM golang:1.18.0-alpine3.15 as builder

ARG VERSION=v0.0.0

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build op-node with the shared go.mod & go.sum files
COPY ./op-node /app/op-node
COPY ./op-chain-ops /app/op-chain-ops
COPY ./op-service /app/op-service
COPY ./op-bindings /app/op-bindings
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
COPY ./.git /app/.git

WORKDIR /app/op-node

RUN make op-node VERSION="$VERSION"

FROM alpine:3.15

COPY --from=builder /app/op-node/bin/op-node /usr/local/bin

CMD ["op-node"]
