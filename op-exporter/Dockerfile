FROM golang:1.18.0-alpine3.15 as builder

COPY ./op-exporter /app

WORKDIR /app/
RUN make build

FROM alpine:3.15
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/op-exporter /usr/local/bin/
ENTRYPOINT ["op-exporter"]
CMD ["--help"]
