FROM golang:1.20.4-alpine3.18 as builder

ARG GITCOMMIT=docker
ARG GITDATE=docker
ARG GITVERSION=docker

RUN apk add make jq git gcc musl-dev linux-headers

COPY ./ufm /app

WORKDIR /app

RUN make ufm

FROM alpine:3.18

COPY ./ufm/entrypoint.sh /bin/entrypoint.sh

RUN apk update && \
    apk add ca-certificates && \
    chmod +x /bin/entrypoint.sh

EXPOSE 8080

VOLUME /etc/ufm

COPY --from=builder /app/bin/ufm /bin/ufm

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD ["/bin/ufm", "/etc/ufm/config.toml"]
