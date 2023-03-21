FROM golang:1.19.0-alpine3.15 as builder

# build from root of repo
COPY ./op-exporter /app

WORKDIR /app/
RUN apk --no-cache add make bash jq git
RUN make build

FROM alpine:3.15
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/op-exporter /usr/local/bin/
ENTRYPOINT ["op-exporter"]
CMD ["--help"]
