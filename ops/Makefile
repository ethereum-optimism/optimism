up: down
	DOCKER_BUILDKIT=1 \
	docker-compose \
		-f docker-compose.yml \
		up --build --detach
.PHONY: up

down:
	docker-compose \
		-f docker-compose.yml \
		down
.PHONY: down

ps:
	docker-compose \
		-f docker-compose.yml \
		ps
.PHONY: ps

up-metrics: down-metrics
	DOCKER_BUILDKIT=1 \
	docker-compose \
		-f docker-compose.yml \
		-f docker-compose-metrics.yml \
		up --build --detach
.PHONY: up-metrics

down-metrics:
	docker-compose \
		-f docker-compose.yml \
		-f docker-compose-metrics.yml \
		down
.PHONY: down-metrics

ps-metrics:
	docker-compose \
		-f docker-compose.yml \
		-f docker-compose-metrics.yml \
		ps
.PHONY: ps