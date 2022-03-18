FROM python:3.8.12-slim-buster

RUN apt-get update && \
	apt-get install -y curl openssh-client git build-essential ca-certificates && \
	curl -sL https://deb.nodesource.com/setup_16.x -o nodesource_setup.sh && \
	bash nodesource_setup.sh && \
	apt-get install -y nodejs && \
	npm i -g yarn && \
	npm i -g depcheck && \
	pip install slither-analyzer
