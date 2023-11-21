FROM debian:buster-slim

ENV DEBIAN_FRONTEND=noninteractive

ARG TARGETPLATFORM

COPY install-prysm.sh /usr/local/bin/install-prysm.sh

RUN apt-get update && \
    apt-get install -y curl

RUN /bin/sh /usr/local/bin/install-prysm.sh && \
    chmod +x /usr/local/bin/prysmctl

CMD ["/usr/local/bin/prysmctl"]