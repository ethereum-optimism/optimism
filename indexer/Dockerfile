FROM --platform=$BUILDPLATFORM golang:1.21.1-alpine3.18 as builder

RUN apk add --no-cache make ca-certificates gcc musl-dev linux-headers git jq bash

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

# build indexer with the shared go.mod & go.sum files
COPY ./indexer /app/indexer
COPY ./op-bindings /app/op-bindings
COPY ./op-service /app/op-service
COPY ./op-node /app/op-node
COPY ./op-chain-ops /app/op-chain-ops

WORKDIR /app/indexer

RUN make indexer

FROM alpine:3.18

COPY --from=builder /app/indexer/indexer /usr/local/bin
COPY --from=builder /app/indexer/indexer.toml /app/indexer/indexer.toml
COPY --from=builder /app/indexer/migrations /app/indexer/migrations

WORKDIR /app

ENV INDEXER_MIGRATIONS_DIR="/app/indexer/migrations"

CMD ["indexer", "index", "--config", "/app/indexer/indexer.toml"]
