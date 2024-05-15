FROM golang:1.21.1-alpine3.18 as builder

RUN apk --no-cache add make

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

COPY ./op-bindings /app/op-bindings
COPY ./op-service /app/op-service
COPY ./op-node /app/op-node
COPY ./op-chain-ops /app/op-chain-ops
WORKDIR /app/op-chain-ops
RUN make celo-dbmigrate celo-migrate

FROM alpine:3.18
RUN apk --no-cache add ca-certificates bash rsync

# RUN addgroup -S app && adduser -S app -G app
# USER app
WORKDIR /app

COPY --from=builder /app/op-chain-ops/bin/celo-dbmigrate /app
COPY --from=builder /app/op-chain-ops/bin/celo-migrate /app
ENV PATH="/app:${PATH}"

ENTRYPOINT ["/app/celo-dbmigrate"]
