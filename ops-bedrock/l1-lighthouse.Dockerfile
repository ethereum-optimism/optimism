ARG TARGETARCH

FROM sigp/lighthouse:v4.6.0-rc.0-${TARGETARCH}-modern-dev

COPY l1-lighthouse-bn-entrypoint.sh /entrypoint-bn.sh
COPY l1-lighthouse-vc-entrypoint.sh /entrypoint-vc.sh

VOLUME ["/db"]
