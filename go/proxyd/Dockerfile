FROM golang:1.17.2-alpine3.14 AS builder

ARG GITCOMMIT=docker
ARG GITDATE=docker
ARG GITVERSION=docker

RUN apk add make jq && \
    mkdir -p /app

WORKDIR /app
COPY go.mod /app
COPY go.sum /app
COPY cmd /app/cmd
COPY *.go /app/
COPY package.json /app
COPY Makefile /app

RUN make proxyd GITCOMMIT=$GITCOMMIT GITDATE=$GITDATE

FROM alpine:3.14.2

EXPOSE 8080

VOLUME /etc/proxyd.toml

COPY --from=builder /app/bin/proxyd /bin/proxyd
CMD ["/bin/proxyd", "/etc/proxyd.toml"]
