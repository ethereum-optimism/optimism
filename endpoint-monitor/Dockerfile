FROM golang:1.21.1-alpine3.18 as builder

RUN apk --no-cache add make jq bash git alpine-sdk

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

COPY ./endpoint-monitor /app/endpoint-monitor
COPY ./op-service /app/op-service
COPY ./op-node /app/op-node

COPY ./.git /app/.git

WORKDIR /app/endpoint-monitor

RUN make build

FROM alpine:3.18
RUN apk --no-cache add ca-certificates

RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app

COPY --from=builder /app/endpoint-monitor/bin/endpoint-monitor /app

ENTRYPOINT ["/app/endpoint-monitor"]
