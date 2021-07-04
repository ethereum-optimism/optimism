ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY batches.sh /opt/optimism/packages/batch-submitter/batches.sh
RUN chmod +x /opt/optimism/packages/batch-submitter/batches.sh && \
    chmod +x /opt/secret2env && \
    apk add --no-cache musl-dev
