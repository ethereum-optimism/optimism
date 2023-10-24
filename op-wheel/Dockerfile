ARG GCP_PROJECT_ID
ARG GCP_ARTIFACT_REPOSITORY
ARG OP_STACK_GO_BUILDER_TAG=latest
ARG OP_STACK_GO_BUILDER=us-docker.pkg.dev/${GCP_PROJECT_ID}/${GCP_ARTIFACT_REPOSITORY}/images/op_stack_go:$OP_STACK_GO_BUILDER_TAG
FROM $OP_STACK_GO_BUILDER as builder
# See "make golang-docker" and /ops/docker/op-stack-go
FROM alpine:3.18

COPY --from=builder /app/op-wheel/bin/op-wheel /usr/local/bin

CMD ["op-wheel"]
