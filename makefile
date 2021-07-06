.ONESHELL:

IMG_VERSION = $(shell cat ./VERSION)
DATE = $(shell date +'%s')

docker-build:
	docker build --progress=plain --build-arg always_upgrade="$(DATE)" -t omgnetwork/vault:latest .
	docker tag omgnetwork/vault:latest omgnetwork/vault:$(IMG_VERSION)

test:
	docker-compose -f docker/docker-compose.yml up

run:
	docker-compose -f docker/lean-docker-compose.yml up

all: docker-build run

clean:
	mv docker/config/entrypoint.sh /tmp/entrypoint.sh
	mv docker/config/vault.hcl /tmp/vault.hcl
	rm -rf docker/config/*
	mv /tmp/entrypoint.sh docker/config/entrypoint.sh
	mv /tmp/vault.hcl docker/config/vault.hcl