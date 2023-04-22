FROM --platform=$BUILDPLATFORM golang:1.19.0-alpine3.15 as builder

ARG VERSION=v0.0.0

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY ./op-challenger /app/op-challenger
COPY ./op-bindings /app/op-bindings
COPY ./op-node /app/op-node
COPY ./op-service /app/op-service
COPY ./op-signer /app/op-signer

WORKDIR /app/op-challenger

RUN go mod download

RUN make op-challenger

FROM alpine:3.15

COPY --from=builder /app/op-challenger/bin/op-challenger /usr/local/bin

CMD ["op-challenger"]
