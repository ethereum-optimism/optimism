FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./batch-submitter /batch-submitter
COPY ./bss-core /bss-core
COPY ./gas-oracle /gas-oracle
COPY ./indexer /indexer
COPY ./l2geth /l2geth
COPY ./l2geth-exporter /l2geth-exporter
COPY ./op-exporter /op-exporter
COPY ./proxyd /proxyd
COPY ./teleportr /teleportr
COPY ./go.work /go.work

RUN cd /gas-oracle && make gas-oracle

FROM alpine:3.15

RUN apk add --no-cache ca-certificates jq curl
COPY --from=builder /gas-oracle/gas-oracle /usr/local/bin/

WORKDIR /usr/local/bin/
ENTRYPOINT ["gas-oracle"]
