ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY wait-for-l1-and-l2.sh /opt/
RUN chmod +x /opt/wait-for-l1-and-l2.sh && \
    chmod +x /opt/secret2env && \
    chmod +x /opt/optimism/packages/omgx/message-relayer-fast/exec/run-message-relayer-fast.js && \
    apk add --no-cache musl-dev
