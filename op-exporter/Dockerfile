FROM golang:1.18.0-alpine3.15 as builder

COPY ./batch-submitter /app/batch-submitter
COPY ./bss-core /app/bss-core
COPY ./gas-oracle /app/gas-oracle
COPY ./indexer /app/indexer
COPY ./l2geth /app/l2geth
COPY ./l2geth-exporter /app/l2geth-exporter
COPY ./op-exporter /app/op-exporter
COPY ./proxyd /app/proxyd
COPY ./teleportr /app/teleportr
COPY ./go.work /go.work

WORKDIR /app/
RUN make build

FROM alpine:3.15
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/op-exporter /usr/local/bin/
ENTRYPOINT ["op-exporter"]
CMD ["--help"]
