FROM --platform=$BUILDPLATFORM golang:1.19.0-alpine3.15 as builder

ARG VERSION=v0.0.0

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build op-proposer with the shared go.mod & go.sum files
COPY ./op-proposer /app/op-proposer
COPY ./op-bindings /app/op-bindings
COPY ./op-node /app/op-node
COPY ./op-service /app/op-service
COPY ./op-signer /app/op-signer
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
COPY ./.git /app/.git

WORKDIR /app/op-proposer

RUN go mod download

ARG TARGETOS TARGETARCH

RUN make op-proposer VERSION="$VERSION" GOOS=$TARGETOS GOARCH=$TARGETARCH

FROM alpine:3.15

COPY --from=builder /app/op-proposer/bin/op-proposer /usr/local/bin

CMD ["op-proposer"]
