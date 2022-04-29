# Build Geth in a stock Go builder container
FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

COPY ./batch-submitter /app/batch-submitter
COPY ./bss-core /app/bss-core
COPY ./gas-oracle /app/gas-oracle
COPY ./indexer /app/indexer
COPY ./l2geth /app/l2geth
COPY ./l2geth-exporter /app/l2geth-exporter
COPY ./op-exporter /app/op-exporter
COPY ./proxyd /app/proxyd
COPY ./teleportr /app/teleportr
COPY ./go.work /app/go.work

WORKDIR /app/l2geth
RUN make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:3.15

RUN apk add --no-cache ca-certificates jq curl
COPY --from=builder /app/l2geth/build/bin/geth /usr/local/bin/

WORKDIR /usr/local/bin/
EXPOSE 8545 8546 8547
COPY ./ops/scripts/geth.sh .
ENTRYPOINT ["geth"]
