FROM --platform=$BUILDPLATFORM golang:1.19.9-alpine3.16 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build indexer with the shared go.mod & go.sum files
COPY ./indexer /app/indexer
COPY ./op-bindings /app/op-bindings
COPY ./op-service /app/op-service
COPY ./op-node /app/op-node
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

COPY ./.git /app/.git

WORKDIR /app/indexer

RUN go mod download

RUN make indexer

FROM alpine:3.16

COPY --from=builder /app/indexer/indexer /usr/local/bin

CMD ["indexer"]
