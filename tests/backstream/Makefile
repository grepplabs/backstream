.DEFAULT_GOAL := build-up
.PHONY: build up down build-up

ROOT_DIR       := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: help
help:
	@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build-up: build up

build: ## docker-compose build
	docker-compose build

up: ## docker-compose up
	docker-compose up --remove-orphans

up-d: ## docker-compose up in detached mode
	docker-compose up --remove-orphans -d

down: ## docker-compose down
	docker-compose down

test-1:
	curl -v -H 'x-backstream-client-id: 4711' http://localhost:8080/test

test-2:
	curl -v -H 'x-backstream-client-id: 4712' http://localhost:8080/test

delay-1:
	curl -v -H 'x-backstream-client-id: 4711' http://localhost:8080/delay/5

delay-2:
	curl -v -H 'x-backstream-client-id: 4712' -H 'x-backstream-request-timeout: 10s' http://localhost:8080/delay/5
