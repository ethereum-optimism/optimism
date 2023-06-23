FROM golang:1.19.9-alpine3.16 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

# build op-heartbeat with local monorepo go modules
COPY ./op-heartbeat /app/op-heartbeat
COPY ./op-node /app/op-node
COPY ./op-service /app/op-service
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

COPY ./.git /app/.git

WORKDIR /app/op-heartbeat

RUN make op-heartbeat

FROM alpine:3.16

COPY --from=builder /app/op-heartbeat/bin/op-heartbeat /usr/local/bin

CMD ["op-heartbeat"]
