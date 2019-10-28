.ONESHELL:

docker-build:
	docker build -t omisego/immutability-vault-ethereum:latest .

run:
	docker-compose -f docker/docker-compose.yml up

all: docker-build run