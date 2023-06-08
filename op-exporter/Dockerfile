FROM golang:1.19.9-alpine3.16 as builder

# build from root of repo
COPY ./op-exporter /app

WORKDIR /app/
RUN apk --no-cache add make bash jq git
RUN make build

FROM alpine:3.16
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/op-exporter /usr/local/bin/
ENTRYPOINT ["op-exporter"]
CMD ["--help"]
