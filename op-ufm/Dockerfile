FROM golang:1.20.4-alpine3.18 as builder

ARG GITCOMMIT=docker
ARG GITDATE=docker
ARG GITVERSION=docker

RUN apk add make jq git gcc musl-dev linux-headers

COPY ./op-ufm /app

WORKDIR /app

RUN make ufm

FROM alpine:3.18

COPY --from=builder /app/entrypoint.sh /bin/entrypoint.sh
COPY --from=builder /app/bin/ufm /bin/ufm

RUN apk update && \
    chmod +x /bin/entrypoint.sh

RUN apk add ca-certificates jq curl bind-tools

VOLUME /etc/ufm

EXPOSE 8080

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD ["/bin/ufm", "/etc/ufm/config.toml"]
