ARG BUILDER_IMAGE=flashbots/op-geth:latest

FROM $BUILDER_IMAGE

RUN apk add --no-cache jq

COPY entrypoint-builder.sh /entrypoint.sh

VOLUME ["/db"]

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
