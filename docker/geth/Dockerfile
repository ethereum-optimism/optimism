## Begin ethereum/client-go
# Build Geth in a stock Go builder container
FROM golang:1.13-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

RUN mkdir /go-ethereum
RUN git clone https://github.com/ethereum-optimism/go-ethereum.git /go-ethereum
RUN cd /go-ethereum && make all

# Pull all binaries into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/* /usr/local/bin/

EXPOSE 8545 8546 8547 30303 30303/udp

## end ethereum/client-go

RUN apk add --no-cache  openssl jq
COPY entrypoint.sh /bin
RUN chmod +x /bin/entrypoint.sh

EXPOSE 9545
ENTRYPOINT ["sh", "/bin/entrypoint.sh"]
