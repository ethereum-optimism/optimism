# use unstable until v5.2.0 got released, which will then also have multi-arch builds that contain
# the minimal spec
FROM sigp/lighthouse:latest-unstable

COPY l1-lighthouse-bn-entrypoint.sh /entrypoint-bn.sh
COPY l1-lighthouse-vc-entrypoint.sh /entrypoint-vc.sh

VOLUME ["/db"]
