FROM --platform=$BUILDPLATFORM golang:1.21.1-alpine3.18 as builder

ARG VERSION=v0.0.0

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

# build op-program with the shared go.mod & go.sum files
COPY ./op-program /app/op-program
COPY ./op-preimage /app/op-preimage
COPY ./op-node /app/op-node
COPY ./op-chain-ops /app/op-chain-ops
COPY ./op-service /app/op-service
COPY ./op-bindings /app/op-bindings
COPY ./.git /app/.git

WORKDIR /app/op-program

ARG TARGETOS TARGETARCH

RUN make op-program VERSION="$VERSION" GOOS=$TARGETOS GOARCH=$TARGETARCH

FROM alpine:3.18

COPY --from=builder /app/op-program/bin/op-program /usr/local/bin

CMD ["op-program"]
