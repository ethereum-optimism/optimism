DOCKER_COMPOSE_CMD := docker-compose \
	-f docker-compose.yml

build:
	DOCKER_BUILDKIT=1 \
	$(DOCKER_COMPOSE_CMD) \
		build
.PHONY: build

up: down
	DOCKER_BUILDKIT=1 \
	$(DOCKER_COMPOSE_CMD) \
		up --build --detach
.PHONY: up

down:
	$(DOCKER_COMPOSE_CMD) \
		down
.PHONY: down

ps:
	$(DOCKER_COMPOSE_CMD) \
		ps
.PHONY: ps

up-metrics: down-metrics
	DOCKER_BUILDKIT=1 \
	$(DOCKER_COMPOSE_CMD) \
		-f docker-compose-metrics.yml \
		up --build --detach
.PHONY: up-metrics

down-metrics:
	$(DOCKER_COMPOSE_CMD) \
		-f docker-compose-metrics.yml \
		down
.PHONY: down-metrics

ps-metrics:
	$(DOCKER_COMPOSE_CMD) \
		-f docker-compose-metrics.yml \
		ps
.PHONY: ps
