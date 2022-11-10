FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./indexer /go/indexer
COPY ./op-bindings /go/op-bindings
COPY ./op-node /go/op-node

COPY ./indexer/docker.go.work /go/go.work

WORKDIR /go/indexer
RUN make

FROM alpine:3.15

COPY --from=builder /go/indexer/indexer /usr/local/bin

CMD ["indexer"]
