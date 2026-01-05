# Variables
BINARY_NAME=auth
DOCKER_COMPOSE=docker compose
TEST_COMPOSE=docker-compose.test.yaml

.PHONY: help build up down restart test test-clean logs

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

## --- PRODUCTION / DEV ---

build: ## Build the production docker images
	$(DOCKER_COMPOSE) build

up: ## Start the production services in the background
	$(DOCKER_COMPOSE) up -d

down: ## Stop and remove all production containers
	$(DOCKER_COMPOSE) down

restart: down up ## Restart the production services

logs: ## Tail the logs of the production services
	$(DOCKER_COMPOSE) logs -f

## --- TESTING ---

test: ## Run the automated test suite
	$(DOCKER_COMPOSE) -f $(TEST_COMPOSE) up --build --abort-on-container-exit --exit-code-from auth-tests --remove-orphans

test-clean: ## Clean up test containers and volumes
	$(DOCKER_COMPOSE) -f $(TEST_COMPOSE) down -v

## --- UTILS ---

clean-cache: ## Clean go cache and docker builder cache
	go clean -cache
	docker builder prune -f

tidy: ## Run go mod tidy
	go mod tidy