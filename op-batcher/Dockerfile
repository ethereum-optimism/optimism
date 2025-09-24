ARG OP_STACK_GO_BUILDER=us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest
FROM $OP_STACK_GO_BUILDER as builder
# See "make golang-docker" and /ops/docker/op-stack-go

FROM alpine:3.20

COPY --from=builder /usr/local/bin/op-batcher /usr/local/bin/op-batcher

EXPOSE 8545 8546 7300 6060 9090

CMD ["op-batcher"]