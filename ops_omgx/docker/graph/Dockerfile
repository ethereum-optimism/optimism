ARG BUILD_IMAGE="graphprotocol/graph-node"
ARG BUILD_IMAGE_VERSION="latest"

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder
COPY secret2env /opt/secret2env
COPY docker-entrypoint.sh /
COPY start /usr/local/bin/start
RUN chmod +x /docker-entrypoint.sh && \
    chmod +x /opt/secret2env && \
    chmod +x /usr/local/bin/start && \
    apt update && \
    apt install -y musl musl-dev musl-tools
ENTRYPOINT ["/docker-entrypoint.sh"]
