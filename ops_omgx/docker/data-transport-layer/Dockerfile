ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY dtl.sh /opt/optimism/packages/data-transport-layer/dtl.sh
RUN chmod +x /opt/optimism/packages/data-transport-layer/dtl.sh && \
    chmod +x /opt/secret2env && \
    apk add --no-cache musl-dev
