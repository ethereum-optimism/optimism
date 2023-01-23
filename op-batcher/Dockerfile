FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build op-batcher with local monorepo go modules
COPY ./op-batcher/docker.go.work /app/go.work
COPY ./op-bindings /app/op-bindings
COPY ./op-node /app/op-node
COPY ./op-proposer /app/op-proposer
COPY ./op-service /app/op-service
COPY ./op-batcher /app/op-batcher
COPY ./op-signer /app/op-signer
COPY ./.git /app/.git

WORKDIR /app/op-batcher

RUN make op-batcher

FROM alpine:3.15

COPY --from=builder /app/op-batcher/bin/op-batcher /usr/local/bin

ENTRYPOINT ["op-batcher"]
