# Makefile for URL Shortener Project

.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=urlshortener

# Docker parameters
DOCKER_COMPOSE=docker-compose

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./...

integration-test:
	$(GOTEST) -v -tags=integration ./tests/integration/...

clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME) coverage.out coverage.html

run: build
	./$(BINARY_NAME)

deps:
	$(GOGET) -v -t ./...
	$(GOMOD) tidy

docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down


coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

vet:
	$(GOCMD) vet ./...

help:
	@echo "Choose a command run:"
	@echo "  make build          Build the binary"
	@echo "  make test           Run tests"
	@echo "  make clean          Clean the build and test artifacts"
	@echo "  make run            Build and run the binary"
	@echo "  make deps           Install dependencies"
	@echo "  make docker-build   Build Docker image"
	@echo "  make docker-up      Start Docker containers"
	@echo "  make docker-down    Stop Docker containers"
	@echo "  make coverage       Generate test coverage report"
	@echo "  make vet            Run go vet for static analysis"

.PHONY: all build test clean run deps docker-build docker-up docker-down lint coverage vet help

make:
	@echo "Running make command"
	@$(MAKE)
