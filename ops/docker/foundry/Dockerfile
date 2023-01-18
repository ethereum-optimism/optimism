from alpine:3.14 as build-environment
WORKDIR /opt
RUN apk add clang lld curl build-base linux-headers git \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs > rustup.sh \
    && chmod +x ./rustup.sh \
    && ./rustup.sh -y
WORKDIR /opt/foundry

# Only diff from upstream docker image is this clone instead
# of COPY. We select a specific commit to use.
RUN git clone https://github.com/foundry-rs/foundry.git . \
    && git checkout 2ff99025abade470a795724c10648c800a41025e

RUN source $HOME/.profile && cargo build --release \
    && strip /opt/foundry/target/release/forge \
    && strip /opt/foundry/target/release/cast \
    && strip /opt/foundry/target/release/anvil

from alpine:3.14 as foundry-client
ENV GLIBC_KEY=https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
ENV GLIBC_KEY_FILE=/etc/apk/keys/sgerrand.rsa.pub
ENV GLIBC_RELEASE=https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.35-r0/glibc-2.35-r0.apk

RUN apk add linux-headers gcompat
RUN wget -q -O ${GLIBC_KEY_FILE} ${GLIBC_KEY} \
    && wget -O glibc.apk ${GLIBC_RELEASE} \
    && apk add glibc.apk --force
COPY --from=build-environment /opt/foundry/target/release/forge /usr/local/bin/forge
COPY --from=build-environment /opt/foundry/target/release/cast /usr/local/bin/cast
COPY --from=build-environment /opt/foundry/target/release/anvil /usr/local/bin/anvil
ENTRYPOINT ["/bin/sh", "-c"]
