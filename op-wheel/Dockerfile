FROM golang:1.19.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

COPY ./op-wheel /app/op-wheel
COPY ./op-node /app/op-node
COPY ./op-chain-ops /app/op-chain-ops
COPY ./op-service /app/op-service
COPY ./op-bindings /app/op-bindings
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app/op-wheel

RUN go build -o op-wheel ./cmd/main.go

FROM alpine:3.15

COPY --from=builder /app/op-wheel/op-wheel /usr/local/bin

CMD ["op-wheel"]
