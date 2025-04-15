#!/bin/bash
# init.sh - Script de inicialização robusto para o API Gateway
# Recomendado para ambientes de produção

set -eo pipefail

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Diretórios e variáveis padrão
CONFIG_PATH=${CONFIG_PATH:-"./config"}
DATA_DIR=${DATA_DIR:-"./data"}
LOG_DIR=${LOG_DIR:-"./logs"}
TEMP_DIR=${TEMP_DIR:-"./tmp"}
PORT=${AG_SERVER_PORT:-8080}
ENV=${ENVIRONMENT:-"development"}
MIN_GO_VERSION="1.23.0"

# Função para logging
log() {
    local level=$1
    local message=$2
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")

    case "$level" in
        "INFO")
            echo -e "${GREEN}[INFO]${NC} ${timestamp} - $message"
            ;;
        "WARN")
            echo -e "${YELLOW}[WARN]${NC} ${timestamp} - $message"
            ;;
        "ERROR")
            echo -e "${RED}[ERROR]${NC} ${timestamp} - $message"
            ;;
        "DEBUG")
            if [ "$ENV" == "development" ]; then
                echo -e "${BLUE}[DEBUG]${NC} ${timestamp} - $message"
            fi
            ;;
    esac
}

# Verifica versão Go usando semver
check_go_version() {
    if ! command -v go >/dev/null; then
        log "ERROR" "Go não encontrado no PATH"
        exit 1
    fi

    local current_version=$(go version | awk '{print $3}' | sed 's/go//')
    log "INFO" "Versão Go encontrada: $current_version"

    # Implementação simplificada de comparação semver
    if [[ "$(printf '%s\n' "$MIN_GO_VERSION" "$current_version" | sort -V | head -n1)" != "$MIN_GO_VERSION" ]]; then
        log "INFO" "Versão Go compatível"
    else
        log "WARN" "Versão Go ($current_version) pode ser inferior à recomendada ($MIN_GO_VERSION)"
    fi
}

# Verifica pré-requisitos
check_prerequisites() {
    log "INFO" "Verificando pré-requisitos..."

    # Verificar se diretórios existem
    for dir in "$CONFIG_PATH" "$DATA_DIR" "$LOG_DIR" "$TEMP_DIR"; do
        if [ ! -d "$dir" ]; then
            log "INFO" "Criando diretório: $dir"
            mkdir -p "$dir"
        fi
    done

    # Verificar se arquivo de configuração existe
    if [ ! -f "${CONFIG_PATH}/config.yaml" ] && [ -z "$AG_DATABASE_DSN" ]; then
        log "ERROR" "Arquivo de configuração não encontrado e variável AG_DATABASE_DSN não definida."
        log "ERROR" "Execute 'make config' ou defina AG_DATABASE_DSN no ambiente."
        exit 1
    fi

    # Verificar se a porta está disponível
    if command -v lsof >/dev/null; then
        if lsof -i :$PORT | grep -q LISTEN; then
            log "ERROR" "Porta $PORT já está em uso."
            exit 1
        fi
    fi

    # Verificar permissões de escrita nos diretórios
    for dir in "$CONFIG_PATH" "$DATA_DIR" "$LOG_DIR" "$TEMP_DIR"; do
        if [ ! -w "$dir" ]; then
            log "ERROR" "Sem permissão de escrita no diretório: $dir"
            exit 1
        fi
    fi

    # Verificar conectividade com serviços externos no modo production
    if [ "$ENV" == "production" ]; then
        # Verificar banco de dados
        if [ ! -z "$AG_DATABASE_DSN" ]; then
            if [[ "$AG_DATABASE_DSN" == *"postgres"* ]]; then
                if ! command -v pg_isready >/dev/null; then
                    log "WARN" "pg_isready não encontrado, pulando verificação de PostgreSQL"
                else
                    # Extrair host e porta do DSN
                    PG_HOST=$(echo $AG_DATABASE_DSN | sed -n 's/.*@\([^:]*\).*/\1/p')
                    PG_PORT=$(echo $AG_DATABASE_DSN | sed -n 's/.*:\([0-9]*\).*/\1/p')
                    PG_PORT=${PG_PORT:-5432}

                    if ! pg_isready -h $PG_HOST -p $PG_PORT; then
                        log "ERROR" "Não foi possível conectar ao PostgreSQL em $PG_HOST:$PG_PORT"
                        exit 1
                    fi
                fi
            fi
        fi

        # Verificar Redis se configurado
        if [ ! -z "$AG_CACHE_REDIS_ADDRESS" ]; then
            REDIS_HOST=$(echo $AG_CACHE_REDIS_ADDRESS | cut -d: -f1)
            REDIS_PORT=$(echo $AG_CACHE_REDIS_ADDRESS | cut -d: -f2)
            REDIS_PORT=${REDIS_PORT:-6379}

            if ! command -v nc >/dev/null; then
                log "WARN" "netcat não encontrado, pulando verificação de Redis"
            else
                if ! nc -z $REDIS_HOST $REDIS_PORT; then
                    log "ERROR" "Não foi possível conectar ao Redis em $REDIS_HOST:$REDIS_PORT"
                    exit 1
                fi
            fi
        fi
    fi

    log "INFO" "Pré-requisitos verificados com sucesso."
}

# Gerar chave JWT se não existir
generate_jwt_key() {
    if [ -z "$AG_AUTH_JWT_SECRET_KEY" ]; then
        log "WARN" "Nenhuma chave JWT encontrada, gerando uma nova..."

        if command -v openssl >/dev/null; then
            export AG_AUTH_JWT_SECRET_KEY=$(openssl rand -base64 64)
        else
            # Fallback para caso openssl não esteja disponível
            export AG_AUTH_JWT_SECRET_KEY=$(head -c 64 /dev/urandom | base64)
        fi

        log "WARN" "Chave JWT gerada temporariamente. Defina AG_AUTH_JWT_SECRET_KEY para persistência."

        # Salvar a chave em arquivo temporário em dev para conveniência
        if [ "$ENV" == "development" ]; then
            echo "$AG_AUTH_JWT_SECRET_KEY" > "$TEMP_DIR/jwt_secret.key"
            log "INFO" "Chave JWT salva em $TEMP_DIR/jwt_secret.key"
        fi
    fi
}

# Aplicar migrações de banco de dados
apply_migrations() {
    log "INFO" "Aplicando migrações de banco de dados..."

    if ! ./apigateway migrate; then
        log "ERROR" "Falha ao aplicar migrações"
        exit 1
    fi

    log "INFO" "Migrações aplicadas com sucesso."
}

# Configurar tratamento de sinais
setup_signal_handlers() {
    log "DEBUG" "Configurando handlers de sinais..."

    trap 'log "INFO" "Recebido sinal SIGTERM/SIGINT, encerrando graciosamente..."; exit' SIGTERM SIGINT
}

# Executar verificação de saúde para confirmar que a aplicação está pronta
check_health() {
    local retries=10
    local wait=2
    local endpoint="http://localhost:$PORT/health/readiness"

    log "INFO" "Verificando disponibilidade da aplicação em $endpoint"

    for i in $(seq 1 $retries); do
        if curl -s -f $endpoint > /dev/null; then
            log "INFO" "Aplicação está disponível e pronta!"
            return 0
        else
            log "DEBUG" "Aguardando aplicação iniciar... ($i/$retries)"
            sleep $wait
        fi
    done

    log "ERROR" "Aplicação não respondeu dentro do tempo esperado"
    return 1
}

# Função principal
main() {
    log "INFO" "Iniciando API Gateway em ambiente '$ENV'"

    # Verificar Go
    check_go_version

    # Verificar pré-requisitos
    check_prerequisites

    # Gerar chave JWT se necessário
    generate_jwt_key

    # Configurar tratamento de sinais
    setup_signal_handlers

    # Aplicar migrações (exceto se explicitamente desabilitado)
    if [ "$SKIP_MIGRATIONS" != "true" ]; then
        apply_migrations
    else
        log "INFO" "Migrações ignoradas conforme configuração"
    fi

    # Iniciar o API Gateway com argumentos adicionais
    log "INFO" "Iniciando API Gateway na porta $PORT..."

    if [ "$ENV" == "development" ]; then
        # Em desenvolvimento, executa sem exec para permitir monitoramento do script
        ./apigateway server --config "$CONFIG_PATH" "$@"
    else
        # Em produção, usa exec para substituir o shell pelo processo da aplicação
        exec ./apigateway server --config "$CONFIG_PATH" "$@"
    fi
}

# Executar a função principal com todos os argumentos passados para o script
main "$@"