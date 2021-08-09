ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY wait-for-l1-and-l2.sh /opt/optimism/packages/omgx/gas-price-oracle/scripts/wait-for-l1-and-l2.sh

RUN chmod +x /opt/optimism/packages/omgx/gas-price-oracle/scripts/wait-for-l1-and-l2.sh && \
    chmod +x /opt/secret2env && \
    apk add --no-cache musl-dev
