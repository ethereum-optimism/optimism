FROM golang:1.18.0-alpine3.15 as builder

COPY ./l2geth-exporter /app

WORKDIR /app/
RUN apk --no-cache add make jq bash git
RUN make build

FROM alpine:3.15
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/l2geth-exporter /usr/local/bin/
ENTRYPOINT ["l2geth-exporter"]
CMD ["--help"]
