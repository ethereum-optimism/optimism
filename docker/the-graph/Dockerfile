FROM rust:latest

# Replace this with the graph-node branch you want to build the image from;
# Note: Docker Hub substitutes this automatically using our hooks/post_checkout script.
ENV SOURCE_BRANCH "master"

# Install clang (required for dependencies)
RUN apt-get update \
    && apt-get install -y clang libclang-dev

# Clone and build the graph-node repository
RUN git clone https://github.com/ethereum-optimism/graph-node.git \
    && cd graph-node \
    && git checkout "$SOURCE_BRANCH" \
    && cargo install --locked --path node \
    && cp docker/start-node /usr/local/bin \
    && cd .. \
    && rm -rf graph-node

# Clone and install wait-for-it
RUN git clone https://github.com/vishnubob/wait-for-it.git \
    && cp wait-for-it/wait-for-it.sh /usr/local/bin \
    && chmod +x /usr/local/bin/wait-for-it.sh \
    && rm -rf wait-for-it

ENV RUST_LOG ""
ENV GRAPH_LOG ""
ENV EARLY_LOG_CHUNK_SIZE ""
ENV ETHEREUM_RPC_PARALLEL_REQUESTS ""
ENV ETHEREUM_BLOCK_CHUNK_SIZE ""

ENV postgres_host ""
ENV postgres_user ""
ENV postgres_pass ""
ENV postgres_db ""
ENV ipfs ""
ENV ethereum ""

# HTTP port
EXPOSE 8000

# WebSocket port
EXPOSE 8001

# JSON-RPC port
EXPOSE 8020

RUN apt-get install gawk

COPY wait-for-ovm.sh /usr/local/bin

# Wait for IPFS and Postgres to start up.
#
# The awk commands below take the IPFS and Postgres and extract
# hostname:port from them. The IPFS port defaults to 443 for HTTPS
# and 80 for HTTP. The Postgres port defaults to 5432.
CMD wait-for-ovm.sh \
    && wait-for-it.sh \
      $(echo $ipfs | \
        gawk 'match($0, /^([a-z]+:\/\/)?([^\/:]+)(:([0-9]+))?.*$/, m) { print m[2]":"(m[4] ? m[4] : (m[1] == "https://" ? 443 : 80)) }') \
      -t 30 \
    && wait-for-it.sh \
         $(echo $postgres_host | \
           gawk 'match($0, /^([a-z]+:\/\/)?([^\/:]+)(:([0-9]+))?.*$/, m) { print m[2]":"(m[4] ? m[4] : 5432) }') \
         -t 30 \
    && sleep 5 \
    && start-node