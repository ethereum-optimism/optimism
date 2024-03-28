FROM --platform=$BUILDPLATFORM golang:1.21.3-alpine3.18 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# We copy the go.mod/sum first, so the `go mod download` does not have to re-run if dependencies do not change.
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN echo "go mod cache: $(go env GOMODCACHE)"
RUN echo "go build cache: $(go env GOCACHE)"

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download

# NOTE: the Dockerfile.dockerignore file effectively describes all dependencies
COPY . /app

ARG GIT_COMMIT
ARG GIT_DATE

ARG DASERVER_VERSION=v0.0.0

ARG TARGETOS TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build cd op-plasma && make da-server  \
    GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$DASERVER_VERSION"

FROM alpine:3.18

COPY --from=builder /app/op-plasma/bin/da-server /usr/local/bin/
