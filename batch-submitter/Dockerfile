FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./batch-submitter /go/batch-submitter
COPY ./bss-core /go/bss-core
COPY ./l2geth /go/l2geth
COPY ./batch-submitter/docker.go.work /go/go.work

WORKDIR /go/batch-submitter
RUN make

FROM alpine:3.15

RUN apk add --no-cache ca-certificates jq curl
COPY --from=builder /go/batch-submitter/batch-submitter /usr/local/bin/

WORKDIR /usr/local/bin
COPY ./ops/scripts/batch-submitter.sh .
ENTRYPOINT ["batch-submitter"]
