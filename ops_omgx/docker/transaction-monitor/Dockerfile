ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY start_service.sh /start_service.sh

RUN chmod +x /start_service.sh && \
    chmod +x /opt/secret2env && \
    apt update && \
    apt install -y musl musl-dev musl-tools
