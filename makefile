.ONESHELL:

DATE = $(shell date +'%s')

docker-build:
	docker build --build-arg always_upgrade="$(DATE)" -t omisego/immutability-vault-ethereum:latest .

run:
	docker-compose -f docker/docker-compose.yml up --build

all: docker-build run