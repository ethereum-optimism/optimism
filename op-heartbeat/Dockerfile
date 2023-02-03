FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build op-heartbeat with local monorepo go modules
COPY ./op-heartbeat /app/op-heartbeat
COPY ./op-node /app/op-node
COPY ./op-service /app/op-service
COPY ./.git /app/.git
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app/op-heartbeat

RUN make op-heartbeat

FROM alpine:3.15

COPY --from=builder /app/op-heartbeat/bin/op-heartbeat /usr/local/bin

CMD ["op-heartbeat"]
