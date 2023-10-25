ARG OP_STACK_GO_BUILDER=us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest
FROM $OP_STACK_GO_BUILDER as builder
# See "make golang-docker" and /ops/docker/op-stack-go
FROM alpine:3.18

COPY --from=builder /app/op-wheel/bin/op-wheel /usr/local/bin

CMD ["op-wheel"]
