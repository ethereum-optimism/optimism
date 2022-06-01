FROM ethereum/client-go:alltools-v1.10.17 as geth

FROM ghcr.io/foundry-rs/foundry:latest as foundry

FROM python:3.8.12-slim-buster

COPY --from=foundry /usr/local/bin/forge /usr/local/bin/forge
COPY --from=foundry /usr/local/bin/cast /usr/local/bin/cast
COPY --from=geth /usr/local/bin/abigen /usr/local/bin/abigen

RUN apt-get update && \
	apt-get install -y curl openssh-client git build-essential ca-certificates musl && \
	curl -sL https://deb.nodesource.com/setup_16.x -o nodesource_setup.sh && \
	curl -sL https://go.dev/dl/go1.18.2.linux-amd64.tar.gz -o go1.18.2.linux-amd64.tar.gz && \
	tar -C /usr/local/ -xzvf go1.18.2.linux-amd64.tar.gz && \
	ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt && \
	bash nodesource_setup.sh && \
	apt-get install -y nodejs && \
	npm i -g yarn && \
	npm i -g depcheck && \
	pip install slither-analyzer