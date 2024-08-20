FROM jinmel/builder:latest

RUN apk add --no-cache jq

COPY entrypoint-builder.sh /entrypoint.sh

VOLUME ["/db"]

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
