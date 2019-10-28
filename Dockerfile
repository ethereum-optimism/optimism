# ****************************************************************************************
# ******** Create a dev environment to build alpine vault plugins and build them *********
# ****************************************************************************************
FROM vault:latest as build

# Setup the alpine build environment for golang
RUN apk add --update alpine-sdk
RUN apk update && apk add go git openssh gcc musl-dev linux-headers

WORKDIR /app

COPY go.mod .
COPY go.sum .

# Get deps - will also be cached if we won't change mod/sum
RUN go mod download

COPY  / .
RUN mkdir -p /app/bin \
	&& GO111MODULE=on CGO_ENABLED=1 GOOS=linux go build -a -i -o /app/bin/immutability-eth-plugin . \
	&& sha256sum -b /app/bin/immutability-eth-plugin > /app/bin/SHA256SUMS

# ***********************************************************
# ********** This is our actual released container **********
# ***********************************************************
FROM vault:latest
USER vault
WORKDIR /app
RUN mkdir -p /home/vault/ca \
    /home/vault/config \
    /home/vault/scripts \
    /home/vault/plugins
# Install the plugin.
COPY --from=build /app/bin/immutability-eth-plugin /home/vault/plugins/immutability-eth-plugin
COPY --from=build /app/bin/SHA256SUMS /home/vault/plugins/SHA256SUMS
