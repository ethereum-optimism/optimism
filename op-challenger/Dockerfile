FROM --platform=$BUILDPLATFORM golang:1.21.1-alpine3.18 as builder

ARG VERSION=v0.0.0

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

# build op-challenger with the shared go.mod & go.sum files
COPY ./op-challenger /app/op-challenger
COPY ./op-program /app/op-program
COPY ./op-preimage /app/op-preimage
COPY ./op-bindings /app/op-bindings
COPY ./op-node /app/op-node
COPY ./op-service /app/op-service
COPY ./.git /app/.git

# Copy cannon and its dependencies
COPY ./cannon /app/cannon
COPY ./op-preimage /app/op-preimage
COPY ./op-chain-ops /app/op-chain-ops

WORKDIR /app/op-program

ARG TARGETOS TARGETARCH

RUN make op-program-host VERSION="$VERSION" GOOS=$TARGETOS GOARCH=$TARGETARCH

WORKDIR /app/cannon

ARG TARGETOS TARGETARCH

RUN make cannon VERSION="$VERSION" GOOS=$TARGETOS GOARCH=$TARGETARCH

WORKDIR /app/op-challenger

RUN make op-challenger VERSION="$VERSION" GOOS=$TARGETOS GOARCH=$TARGETARCH

FROM alpine:3.18

# Make the bundled op-program the default cannon server
COPY --from=builder /app/op-program/bin/op-program /usr/local/bin
ENV OP_CHALLENGER_CANNON_SERVER /usr/local/bin/op-program

# Make the bundled cannon the default cannon executable
COPY --from=builder /app/cannon/bin/cannon /usr/local/bin
ENV OP_CHALLENGER_CANNON_BIN /usr/local/bin/cannon

COPY --from=builder /app/op-challenger/bin/op-challenger /usr/local/bin

CMD ["op-challenger"]
