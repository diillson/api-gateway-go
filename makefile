# Makefile para o API Gateway
# Facilita tarefas comuns de desenvolvimento e operação

# Variáveis
BINARY_NAME=apigateway
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH)"
DOCKER_IMAGE=api-gateway-go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint
GOTOOL=$(GOCMD) tool

# Variáveis de ambiente
ENV ?= development
CONFIG_PATH ?= ./config

.PHONY: all build clean test lint deps docker-build docker-run help config init \
		run-dev run-prod migrate create-admin generate-token coverage profile bench

# Alvo padrão
all: lint test build

# Compilar a aplicação
build:
	@echo "Compilando para $(ENV)..."
	@CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/apigateway

# Compilação para desenvolvimento com símbolos de debug
build-dev:
	@echo "Compilando versão de desenvolvimento..."
	@$(GOBUILD) $(LDFLAGS) -race -gcflags="all=-N -l" -o $(BINARY_NAME) ./cmd/apigateway

# Limpar artefatos de compilação
clean:
	@echo "Limpando..."
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out

# Executar testes
test:
	@echo "Executando testes..."
	@$(GOTEST) -v ./...

# Executar testes com cobertura
coverage:
	@echo "Executando testes com cobertura..."
	@$(GOTEST) -cover -coverprofile=coverage.out ./...
	@$(GOTOOL) cover -html=coverage.out

# Executar benchmark
bench:
	@echo "Executando benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

# Verificar código com linter
lint:
	@echo "Executando linter..."
	@$(GOLINT) run --deadline=5m

# Gerenciar dependências
deps:
	@echo "Verificando dependências..."
	@$(GOMOD) tidy
	@$(GOMOD) verify

# Gerar arquivo de configuração
config:
	@echo "Gerando arquivo de configuração para $(ENV)..."
	@mkdir -p $(CONFIG_PATH)
	@go run cmd/genconfig/main.go --output $(CONFIG_PATH)/config.yaml

# Inicializar a aplicação (usa o script init.sh)
init:
	@echo "Inicializando API Gateway em ambiente $(ENV)..."
	@ENV=$(ENV) CONFIG_PATH=$(CONFIG_PATH) ./init.sh

# Executar aplicação em modo de desenvolvimento
run-dev: build-dev
	@echo "Executando em modo desenvolvimento..."
	@ENV=development CONFIG_PATH=$(CONFIG_PATH) ./$(BINARY_NAME) server

# Executar aplicação em modo de produção
run-prod: build
	@echo "Executando em modo produção..."
	@ENV=production CONFIG_PATH=$(CONFIG_PATH) ./$(BINARY_NAME) server

# Aplicar migrações de banco de dados
migrate:
	@echo "Aplicando migrações..."
	@go run cmd/migrate/main.go

# Criar uma nova migração
create-migration:
	@read -p "Nome da migração: " name; \
	go run cmd/migrate/main.go -action=create -name=$$name

# Criar usuário administrador
create-admin:
	@echo "Criando usuário administrador..."
	@read -p "Username: " username; \
	read -p "Email: " email; \
	read -s -p "Password: " password; \
	echo ""; \
	go run cmd/tools/create_admin.go -username=$$username -email=$$email -password=$$password

# Gerar token JWT para um usuário
generate-token:
	@echo "Gerando token JWT..."
	@read -p "User ID: " user_id; \
	go run cmd/tools/generate_token.go -user_id=$$user_id

# Executar com profiling ativado
profile:
	@echo "Executando com profiler em http://localhost:6060/debug/pprof/"
	@GO_PROFILE=true ./$(BINARY_NAME) server

# Iniciar apenas o Zipkin para traces
start-zipkin:
	@echo "Iniciando Zipkin na porta 9411..."
	@docker-compose up -d zipkin

# Parar o Zipkin
stop-zipkin:
	@echo "Parando Zipkin..."
	@docker-compose stop zipkin

# Abrir UI do Zipkin no navegador
open-zipkin:
	@echo "Abrindo Zipkin UI no navegador..."
	@xdg-open http://localhost:9411 2>/dev/null || open http://localhost:9411 2>/dev/null || echo "Abra manualmente: http://localhost:9411"

# Construir imagem Docker
docker-build:
	@echo "Construindo imagem Docker $(DOCKER_IMAGE):$(VERSION)..."
	@docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Executar em Docker
docker-run:
	@echo "Executando container Docker..."
	@docker run -p 8080:8080 --name api-gateway -e ENV=$(ENV) $(DOCKER_IMAGE):latest

# Docker Compose para ambiente de desenvolvimento completo
docker-dev:
	@echo "Iniciando ambiente de desenvolvimento com docker-compose..."
	@docker-compose -f docker-compose.yml up -d

# Exibir ajuda
help:
	@echo "API Gateway - Comandos disponíveis:"
	@echo ""
	@echo "  make build         - Compila a aplicação"
	@echo "  make build-dev     - Compila com flags de desenvolvimento"
	@echo "  make clean         - Remove artefatos de compilação"
	@echo "  make test          - Executa testes unitários"
	@echo "  make coverage      - Executa testes com relatório de cobertura"
	@echo "  make bench         - Executa benchmarks"
	@echo "  make lint          - Verifica o código com linter"
	@echo "  make deps          - Verifica e atualiza dependências"
	@echo "  make config        - Gera arquivo de configuração padrão"
	@echo "  make init          - Inicializa o ambiente e roda a aplicação"
	@echo "  make run-dev       - Executa em modo desenvolvimento"
	@echo "  make run-prod      - Executa em modo produção"
	@echo "  make migrate       - Aplica migrações pendentes"
	@echo "  make create-migration - Cria uma nova migração"
	@echo "  make create-admin  - Cria usuário administrador"
	@echo "  make generate-token - Gera token JWT para um usuário"
	@echo "  make profile       - Executa com profiling ativado"
	@echo "  make docker-build  - Constrói imagem Docker"
	@echo "  make docker-run    - Executa container Docker"
	@echo "  make docker-dev    - Inicia ambiente completo com docker-compose"
	@echo ""
	@echo "Variáveis:"
	@echo "  ENV          - Ambiente (development, staging, production)"
	@echo "  CONFIG_PATH  - Caminho para arquivos de configuração"
