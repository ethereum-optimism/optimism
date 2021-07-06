ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY geth.sh /geth.sh
RUN chmod +x /geth.sh && \
    chmod +x /opt/secret2env && \
    apk add --no-cache musl-dev
EXPOSE 8545 8546
