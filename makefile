# Makefile

# Variáveis
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=apigateway
MAIN_FILE=cmd/apigateway/main.go
DOCKER_COMPOSE=docker-compose
K6=k6

# Cores para saída
GREEN=\033[0;32m
NC=\033[0m # No Color

# Targets
.PHONY: all test clean build run docker test-unit test-integration test-security test-load

all: test build

build:
    @echo "${GREEN}Building API Gateway...${NC}"
    $(GOBUILD) -o $(BINARY_NAME) $(MAIN_FILE)

clean:
    @echo "${GREEN}Cleaning up...${NC}"
    $(GOCLEAN)
    rm -f $(BINARY_NAME)

run: build
    @echo "${GREEN}Running API Gateway...${NC}"
    ./$(BINARY_NAME) server

test: test-unit test-integration

test-unit:
    @echo "${GREEN}Running unit tests...${NC}"
    $(GOTEST) -v ./internal/...

test-integration:
    @echo "${GREEN}Running integration tests...${NC}"
    $(GOTEST) -v ./tests/integration/...

    test-security:
        @echo "${GREEN}Running security tests...${NC}"
        $(GOTEST) -v ./tests/security/...

test-load:
    @echo "${GREEN}Running load tests...${NC}"
    $(K6) run tests/load/basic_load_test.js

test-all: test-unit test-integration test-security test-load
    @echo "${GREEN}All tests completed!${NC}"

coverage:
    @echo "${GREEN}Generating test coverage...${NC}"
    $(GOTEST) -cover -coverprofile=coverage.out ./...
    $(GOCMD) tool cover -html=coverage.out

docker-build:
    @echo "${GREEN}Building Docker image...${NC}"
    docker build -t api-gateway:latest .

docker-run: docker-build
    @echo "${GREEN}Running Docker container...${NC}"
    docker run -p 8080:8080 api-gateway:latest

docker-compose-up:
    @echo "${GREEN}Starting Docker Compose environment...${NC}"
    $(DOCKER_COMPOSE) up -d

docker-compose-down:
    @echo "${GREEN}Stopping Docker Compose environment...${NC}"
    $(DOCKER_COMPOSE) down

lint:
    @echo "${GREEN}Running linters...${NC}"
    golangci-lint run ./...

generate-mocks:
    @echo "${GREEN}Generating mocks...${NC}"
    mockery --dir=internal/domain/repository --name=RouteRepository --output=internal/mocks
    mockery --dir=pkg/cache --name=Cache --output=internal/mocks
    mockery --dir=internal/app/auth --name=AuthService --output=internal/mocks

# Helper para instalar dependências de desenvolvimento
dev-setup:
    @echo "${GREEN}Installing development dependencies...${NC}"
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install github.com/vektra/mockery/v2@latest