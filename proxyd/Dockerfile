FROM golang:1.18.0-alpine3.15 as builder

ARG GITCOMMIT=docker
ARG GITDATE=docker
ARG GITVERSION=docker

RUN apk add make jq git gcc musl-dev linux-headers

WORKDIR /app

COPY ./batch-submitter /app/
COPY ./bss-core /app/
COPY ./gas-oracle /app/
COPY ./indexer /app/
COPY ./l2geth /app/
COPY ./l2geth-exporter /app/
COPY ./op-exporter /app/
COPY ./proxyd /app/
COPY ./teleportr /app/
COPY ./go.work /app/go.work

RUN make proxyd

FROM alpine:3.15

COPY ./proxyd/entrypoint.sh /bin/entrypoint.sh

RUN apk update && \
    apk add ca-certificates && \
    chmod +x /bin/entrypoint.sh

EXPOSE 8080

VOLUME /etc/proxyd

COPY --from=builder /app/bin/proxyd /bin/proxyd

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD ["/bin/proxyd", "/etc/proxyd/proxyd.toml"]
