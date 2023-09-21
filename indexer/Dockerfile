FROM --platform=$BUILDPLATFORM golang:1.20.7-alpine3.18 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

# build indexer with the shared go.mod & go.sum files
COPY ./indexer /app/indexer
COPY ./op-bindings /app/op-bindings
COPY ./op-service /app/op-service
COPY ./op-node /app/op-node

WORKDIR /app/indexer

RUN make indexer

FROM alpine:3.18

COPY --from=builder /app/indexer/indexer /usr/local/bin

CMD ["indexer", "index", "--config", "/app/indexer/indexer.toml"]
