FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./batch-submitter /go/batch-submitter
COPY ./bss-core /go/bss-core
COPY ./gas-oracle /go/gas-oracle
COPY ./indexer /go/indexer
COPY ./l2geth /go/l2geth
COPY ./l2geth-exporter /go/l2geth-exporter
COPY ./op-exporter /go/op-exporter
COPY ./proxyd /go/proxyd
COPY ./teleportr /go/teleportr
COPY ./go.work /go/go.work

WORKDIR /go/teleportr
RUN make teleportr teleportr-api

FROM alpine:3.15

RUN apk add --no-cache ca-certificates jq curl
COPY --from=builder /go/teleportr/teleportr /usr/local/bin/
COPY --from=builder /go/teleportr/teleportr-api /usr/local/bin/

CMD ["teleportr"]
