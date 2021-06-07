ARG BUILD_IMAGE=""
ARG BUILD_IMAGE_VERSION=""

FROM ${BUILD_IMAGE}:${BUILD_IMAGE_VERSION} as builder

COPY secret2env /opt/secret2env
COPY wait-for-l1.sh /wait-for-l1.sh
RUN cp -fRv /opt/contracts/dist/dumps/* /opt/contracts/build
RUN chmod +x /wait-for-l1.sh && \
    chmod +x /opt/secret2env && \
    apt-get update && \
    apt-get install -y musl musl-dev musl-tools
