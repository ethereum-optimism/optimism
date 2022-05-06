# Build Geth in a stock Go builder container
FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

COPY ./l2geth /app/l2geth

WORKDIR /app/l2geth
RUN make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:3.15

RUN apk add --no-cache ca-certificates jq curl
COPY --from=builder /app/l2geth/build/bin/geth /usr/local/bin/

WORKDIR /usr/local/bin/
EXPOSE 8545 8546 8547
COPY ./ops/scripts/geth.sh .
ENTRYPOINT ["geth"]
