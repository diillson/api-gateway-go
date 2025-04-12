#!/bin/bash
# startup.sh - Script de inicialização seguro para o API Gateway

set -e

# Carregar variáveis de ambiente, se disponíveis
if [ -f ".env" ]; then
   source .env
fi

# Definir variáveis padrão
CONFIG_PATH=${CONFIG_PATH:-"./config"}
DATA_DIR=${DATA_DIR:-"./data"}
LOG_DIR=${LOG_DIR:-"./logs"}
PORT=${AG_SERVER_PORT:-8080}

# Função para validar pré-requisitos
check_prerequisites() {
    echo "Verificando pré-requisitos..."

# Verificar se diretórios existem
    for dir in "$CONFIG_PATH" "$DATA_DIR" "$LOG_DIR"; do
        if [ ! -d "$dir" ]; then
           echo "Criando diretório: $dir"
           mkdir -p "$dir"
        fi
    done

    # Verificar se arquivo de configuração existe
    if [ ! -f "${CONFIG_PATH}/config.yaml" ] && [ -z "$AG_DATABASE_DSN" ]; then
        echo "ERRO: Arquivo de configuração não encontrado e variável AG_DATABASE_DSN não definida."
        echo "Execute ./apigateway genconfig ou defina AG_DATABASE_DSN no ambiente."
        exit 1
    fi

    # Verificar se porta está disponível
    if command -v lsof >/dev/null; then
        if lsof -i :$PORT | grep -q LISTEN; then
            echo "ERRO: Porta $PORT já está em uso."
            exit 1
        fi
    fi

    echo "Pré-requisitos verificados com sucesso."
}

# Função para gerar chave JWT se não existir
generate_jwt_key() {
    if [ -z "$AG_AUTH_JWTSECRET" ]; then
        echo "Nenhuma chave JWT encontrada, gerando uma nova..."
        export AG_AUTH_JWTSECRET=$(openssl rand -base64 32)
        echo "AVISO: Chave JWT gerada temporariamente. Defina AG_AUTH_JWTSECRET para persistência."
    fi
}

# Função para aplicar migrações de banco de dados
apply_migrations() {
    echo "Aplicando migrações de banco de dados..."
    ./apigateway migrate
    echo "Migrações aplicadas com sucesso."
}

# Verificar pré-requisitos
check_prerequisites

# Gerar chave JWT se necessário
generate_jwt_key

# Aplicar migrações
apply_migrations

# Iniciar o API Gateway
echo "Iniciando API Gateway na porta $PORT..."
exec ./apigateway server --config "$CONFIG_PATH"