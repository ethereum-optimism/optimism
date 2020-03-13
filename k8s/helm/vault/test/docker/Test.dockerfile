# This Dockerfile installs all the dependencies necessary to run the unit and
# acceptance tests. This image also contains gcloud so you can run tests
# against a GKE cluster easily.
#
# This image has no automatic entrypoint. It is expected that you'll run
# a script to configure kubectl, potentially install Helm, and run the tests
# manually. This image only has the dependencies pre-installed.

FROM alpine:latest
WORKDIR /root

ENV BATS_VERSION "1.1.0"

# base packages
RUN apk update && apk add --no-cache --virtual .build-deps \
    ca-certificates \
    curl \
    tar \
    bash \
    openssl \
    python \
    py-pip \
    git \
    jq

# yq
RUN pip install yq

# gcloud
RUN curl -OL https://dl.google.com/dl/cloudsdk/channels/rapid/install_google_cloud_sdk.bash && \
    bash install_google_cloud_sdk.bash --disable-prompts --install-dir='/root/' && \
    ln -s /root/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud

# kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# helm
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

# bats
RUN curl -sSL https://github.com/bats-core/bats-core/archive/v${BATS_VERSION}.tar.gz -o /tmp/bats.tgz \
    && tar -zxf /tmp/bats.tgz -C /tmp \
    && /bin/bash /tmp/bats-core-$BATS_VERSION/install.sh /usr/local
