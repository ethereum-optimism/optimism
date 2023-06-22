FROM golang:1.19.9-alpine3.16 as builder

RUN apk --no-cache add make jq bash git alpine-sdk

COPY ./endpoint-monitor /app/endpoint-monitor
COPY ./op-service /app/op-service
COPY ./op-node /app/op-node
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

COPY ./.git /app/.git

WORKDIR /app/endpoint-monitor

RUN go mod download

RUN make build

FROM alpine:3.16
RUN apk --no-cache add ca-certificates

RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app

COPY --from=builder /app/endpoint-monitor/bin/endpoint-monitor /app

ENTRYPOINT ["/app/endpoint-monitor"]
