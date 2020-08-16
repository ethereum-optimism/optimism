# ****************************************************************************************
# ******** Create a dev environment to build alpine vault plugins and build them *********
# ****************************************************************************************

FROM golang:1.14-alpine as build
# Setup the alpine build environment for golang
RUN apk add --update alpine-sdk
RUN apk update && apk add git openssh gcc musl-dev linux-headers

WORKDIR /app

COPY go.mod .
COPY go.sum .

# Get deps - will also be cached if we won't change mod/sum
RUN go version
RUN go mod download

COPY  / .
RUN mkdir -p /app/bin \
    && CGO_ENABLED=1 GOOS=linux go build -a -i -o /app/bin/immutability-eth-plugin . \
    && sha256sum -b /app/bin/immutability-eth-plugin > /app/bin/SHA256SUMS

# ***********************************************************
# ********** This is our actual released container **********
# ***********************************************************
FROM vault:latest
# we pass epoch time so it always upgrades
ARG always_upgrade
RUN echo ${always_upgrade} > /dev/null && apk update && apk upgrade
RUN apk add bash openssl jq
USER vault
WORKDIR /app
RUN mkdir -p /home/vault/ca \
    /home/vault/config \
    /home/vault/scripts \
    /home/vault/plugins
# Install the plugin.
COPY --from=build /app/bin/immutability-eth-plugin /home/vault/plugins/immutability-eth-plugin
COPY --from=build /app/bin/SHA256SUMS /home/vault/plugins/SHA256SUMS
HEALTHCHECK CMD nc -zv 127.0.0.1 8900 || exit 1
