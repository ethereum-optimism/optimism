FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./indexer /go/indexer
COPY ./.git /go/.git
COPY ./indexer/go.mod /go/indexer/go.mod
COPY ./indexer/go.sum /go/indexer/go.sum


WORKDIR /go/indexer
RUN make

FROM alpine:3.15

COPY --from=builder /go/indexer/indexer /usr/local/bin

CMD ["indexer"]
