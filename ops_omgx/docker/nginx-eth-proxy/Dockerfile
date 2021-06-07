FROM openresty/openresty:buster
LABEL maintainer="Optimistic Systems <systems@optiomism.io>"
ARG GOTEMPLATE_VERSION=v3.9.0

RUN DEBIAN_FRONTEND=noninteractive apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        openresty-opm \
    && opm get knyar/nginx-lua-prometheus

RUN curl -o /usr/local/bin/gomplate \
    -sSL https://github.com/hairyhenderson/gomplate/releases/download/$GOTEMPLATE_VERSION/gomplate_linux-amd64-slim \
    && chmod +x /usr/local/bin/gomplate

RUN mkdir -p /var/log/nginx/ \
    && ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log

COPY eth-jsonrpc-access.lua /usr/local/openresty/nginx/eth-jsonrpc-access.lua
COPY nginx.template.conf /docker-entrypoint.d/nginx.template.conf
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]
