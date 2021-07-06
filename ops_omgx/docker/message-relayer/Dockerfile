ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY wait-for-l1-and-l2.sh /opt/
COPY relayer.sh /opt/optimism/packages/message-relayer/relayer.sh
RUN chmod +x /opt/wait-for-l1-and-l2.sh && \
    chmod +x /opt/optimism/packages/message-relayer/relayer.sh && \
    chmod +x /opt/secret2env && \
    chmod +x /opt/optimism/packages/message-relayer/exec/run-message-relayer.js && \
    apk add --no-cache musl-dev
