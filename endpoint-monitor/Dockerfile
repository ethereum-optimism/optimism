FROM golang:1.18.0-alpine3.15 as builder

COPY ./endpoint-monitor /app

WORKDIR /app
RUN apk --no-cache add make jq bash git alpine-sdk
RUN make build

FROM alpine:3.15
RUN apk --no-cache add ca-certificates

RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app

COPY --from=builder /app/bin/endpoint-monitor /app

ENTRYPOINT ["/app/endpoint-monitor"]
