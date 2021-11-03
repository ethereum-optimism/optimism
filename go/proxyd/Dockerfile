FROM golang:1.17.2-alpine3.13 AS builder

ARG GITCOMMIT=docker
ARG GITDATE=docker
ARG GITVERSION=docker

RUN apk add make jq git

WORKDIR /app
COPY ./go/proxyd /app

RUN make proxyd

FROM alpine:3.14.2

EXPOSE 8080

VOLUME /etc/proxyd

COPY --from=builder /app/bin/proxyd /bin/proxyd
CMD ["/bin/proxyd", "/etc/proxyd/proxyd.toml"]
