FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

COPY ./op-wheel/docker.go.work /app/go.work
COPY ./op-node /app/op-node
COPY ./op-service /app/op-service
COPY ./op-wheel /app/op-wheel

WORKDIR /app/op-wheel

RUN go build -o op-wheel ./cmd/main.go

FROM alpine:3.15

COPY --from=builder /app/op-wheel/op-wheel /usr/local/bin

CMD ["op-wheel"]
