ARG GCP_PROJECT_ID
ARG GCP_ARTIFACT_REPOSITORY
ARG OP_STACK_GO_BUILDER_TAG=latest
ARG OP_STACK_GO_BUILDER=us-docker.pkg.dev/${GCP_PROJECT_ID}/${GCP_ARTIFACT_REPOSITORY}/images/op_stack_go:$OP_STACK_GO_BUILDER_TAG
FROM $OP_STACK_GO_BUILDER as builder
# See "make golang-docker" and /ops/docker/op-stack-go

FROM alpine:3.18

COPY --from=builder /usr/local/bin/op-program /usr/local/bin/op-program

CMD ["op-program"]
