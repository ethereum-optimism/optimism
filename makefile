.ONESHELL:

IMG_VERSION = $(shell cat ./VERSION)
DATE = $(shell date +'%s')

docker-build:
	docker build --build-arg always_upgrade="$(DATE)" -t omgnetwork/vault:latest .
	docker tag omgnetwork/vault:latest omgnetwork/vault:$(IMG_VERSION)

test:
	docker-compose -f docker/docker-compose.yml up

run:
	docker-compose -f docker/lean-docker-compose.yml up

all: docker-build run