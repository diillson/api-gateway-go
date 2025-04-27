# API Gateway em Go

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/diillson/api-gateway-go)
![Build Status](https://img.shields.io/github/actions/workflow/status/diillson/api-gateway-go/ci.yml?branch=main)
![Coverage](https://img.shields.io/codecov/c/github/diillson/api-gateway-go)
![License](https://img.shields.io/github/license/diillson/api-gateway-go)

Um API Gateway robusto, escalável e de alto desempenho escrito em Go. Ideal para arquiteturas de microsserviços, fornecendo recursos como autenticação, rate limiting, circuit breaking, métricas e muito mais.

## 🌟 Recursos
    
- **Proxy Reverso**: Encaminha requisições para serviços de backend.
- **Autenticação**: Valida tokens JWT e controla acesso a rotas protegidas.
- **Rate Limiting**: Limita o número de requisições por usuário, IP ou rota.
- **Circuit Breaking**: Evita sobrecarga de serviços downstream quando falham.
- **Caching**: Reduz carga em serviços repetindo respostas previamente obtidas.
- **Monitoramento**: Métricas detalhadas via Prometheus e dashboards Grafana.
- **Rastreamento**: Rastreamento distribuído com OpenTelemetry.
- **Admin API**: Interface para gerenciar rotas e configurações.
- **Escalabilidade**: Projetado para alta performance e baixo consumo de recursos.
    
## 📋 Pré-requisitos
    
- Go 1.23 ou superior
- Docker e Docker Compose (para desenvolvimento e teste)
- Banco de dados compatível (SQLite, PostgreSQL ou MySQL)
- Redis (opcional, para cache distribuído)
    
## 🚀 Início Rápido
    
### Instalação Básica
    
```bash
  # Clonar o repositório
  git clone https://github.com/diillson/api-gateway-go.git
  cd api-gateway-go
    
  # Instalar dependências
  go mod download
    
  # Gerar arquivo de configuração
  go run cmd/genconfig/main.go --output config/config.yaml
    
  # Executar migrações
  go run cmd/apigateway/main.go migrate
  
  # Usando a ferramenta CLI incluída para criar admin
  go run cmd/tools/create_admin.go -username "admin" -password "senha123" --email "admin@example.com" -driver postgres -dsn "postgres://postgres:postgres@localhost:5432/apigateway?sslmode=disable"
    
  # Gerar token para acesso administrativo
  # Usando a ferramenta CLI incluída para gerar token
  go run cmd/tools/generate_token.go -user_id "ID GERADO AO CRIAR O USUÁRIO"
    
  # Iniciar o servidor
  go run cmd/apigateway/main.go server
```

### Usando Docker Compose
```bash
    # Iniciar ambiente completo (API Gateway, PostgreSQL, Redis, Prometheus, Grafana)
    docker-compose up -d
    
    # Verificar logs
    docker-compose logs -f api-gateway
    
    # Parar todos os serviços
    docker-compose down
```

# 🤓 O API GATEWAY NO DETALHE 🤩 

## ⚙️ Configuração

O API Gateway pode ser configurado através de:

1. Arquivo de configuração YAML
2. Variáveis de ambiente (prefixo  AG_ )
3. Flags de linha de comando

### Exemplo de Configuração
```yaml
    server:
       port: 8080  # Porta HTTP para o servidor (Não usada para ENV != development e TLS = true)
       host: "0.0.0.0"
       readTimeout: "5s"
       writeTimeout: "10s"
       idleTimeout: "30s"
       maxheaderbytes: 1048576
       tls: false
       certfile: /path/to/cert.pem
       keyfile: /path/to/key.pem
       baseurl: https://api.example.com
       domains:
         - api.example.com

    database:
       driver: postgres             # Opções: sqlite, postgres, mysql
       dsn: postgres://postgres:postgres@postgres:5432/apigateway?sslmode=disable    # Formato DSN específico para cada driver
       maxIdleConns: 10
       maxOpenConns: 50
       connMaxLifetime: "1h"
       loglevel: warn
       slowthreshold: 200ms
       migrationdir: ./migrations
      # skipmigrations: true (Apenas usar se for pular as migrações pois default é false)       

    cache:
       enabled: true
       type: "memory"                # Opções: memory, redis
       ttl: "5m"                     # Tempo de vida padrão para itens no cache
    maxitems: 10000
    maxmemorymb: 100
    redis:  # Configurações específicas para Redis
      address: localhost:6379  # Endereço do servidor Redis (host:port)
      password: ""
      db: 0
      poolsize: 10  # Número máximo de conexões no pool
      minidleconns: 5  # Número mínimo de conexões ociosas mantidas abertas
      maxretries: 3  # Número máximo de tentativas de reconexão
      readtimeout: 3s  # Timeout para operações de leitura
      writetimeout: 3s  # Timeout para operações de escrita
      dialtimeout: 5s  # Timeout para estabelecer nova conexão
      pooltimeout: 4s  # Timeout para obter conexão do pool
      idletimeout: 5m0s  # Tempo máximo que uma conexão pode ficar ociosa
      maxconnage: 30m0s  # Tempo máximo de vida da conexão
      connectionpoolname: ""   

    auth:
       enabled: true
       jwtsecret: "your-secret-key"  # Em produção, use variável de ambiente
       tokenExpiration: "24h"
       refreshEnabled: true
       refreshDuration: "168h"
       adminUsers: ["admin"]
       passwordminlen: 8

    logging:
      level: info
      format: json
      outputpath: stdout
      errorpath: stderr
      production: true

    metrics:
      enabled: true
      prometheuspath: "/metrics"
      reportInterval: "15s"
      
    tracing:
      enabled: true
      provider: otlp
      endpoint: otel-collector:4317
      servicename: api-gateway
      samplingratio: 1.0      

    features:
       ratelimiter: true             # Ativar limitação de taxa
       circuitbreaker: true          # Ativar circuit breaker
       caching: true                 # Ativar cache de respostas
       healthcheck: true             # Ativar endpoints de health check
       adminapi: true                # Ativar API administrativa
       monitoring: true              # Ativar monitoramento
```

### Variáveis de Ambiente
```bash

    # Configurações do servidor
    AG_SERVER_PORT=8080
    AG_SERVER_HOST=0.0.0.0
    
    # Configurações do banco de dados
    AG_DATABASE_DRIVER=postgres
    AG_DATABASE_DSN=postgres://user:password@localhost:5432/apigateway
    
    # Configurações de cache
    AG_CACHE_TYPE=redis
    AG_CACHE_ADDRESS=localhost:6379
    
    # Configurações de autenticação (importante!)
    AG_AUTH_JWT_SECRET_KEY=seu-segredo-seguro-aqui
    AG_AUTH_TOKENEXPIRATION=24h
    
    # Ativar/Desativar recursos
    AG_FEATURES_RATELIMITER=true
    AG_FEATURES_CIRCUITBREAKER=true
    AG_FEATURES_CACHING=true
    
    AG_SERVER_TLS=true                          # Habilitar TLS/HTTPS
    SERVER_DOMAINS=api.seudominio.com,outro.seudominio.com  # Domínios para Let's Encrypt
    LETSENCRYPT_EMAIL=seu@email.com             # Email para Let's Encrypt
    AG_SERVER_CERT_FILE=/path/to/cert.pem       # Opcional: Caminho para certificado
    AG_SERVER_KEY_FILE=/path/to/key.pem         # Opcional: Caminho para chave privada
```
## 🔒 Autenticação e Segurança

### Gerando um Token JWT para Acesso Administrativo

Para acessar a área administrativa, você precisa gerar um usuário Admin e um token JWT válido:
```bash
    # Usando a ferramenta CLI incluída para gerar admin
    go run cmd/tools/create_admin.go -username "admin" -password "senha123" --email "admin@example.com" -driver postgres -dsn "postgres://postgres:postgres@localhost:5432/apigateway?sslmode=disable"
    
    # Usando a ferramenta CLI incluída para gerar token
    go run cmd/tools/generate_token.go -user_id "ID GERADO AO CRIAR O USUÁRIO"
```

## 🔒 JWT API Gateway no Detalhe

### Configurando o Segredo JWT
    
    O segredo JWT é usado para assinar e verificar tokens de autenticação. É crucial configurá-lo corretamente para segurança.
    
**Opções para configurar o segredo JWT (em ordem de prioridade):**
    
1. **Via variável de ambiente:**
 ```bash
       export JWT_SECRET_KEY=sua-chave-secreta-muito-longa-e-aleatoria
```
2. Via variável de ambiente com prefixo AG:
```bash
   export AG_AUTH_JWT_SECRET_KEY=sua-chave-secreta-muito-longa-e-aleatoria
```

3. No arquivo de configuração  config.yaml :
```yaml
auth:
   jwtsecret: "sua-chave-secreta-muito-longa-e-aleatoria"
```

⚠️ Importante: O uso do valor padrão hardcoded é apenas para desenvolvimento. Em ambientes de produção, sempre configure um segredo único e seguro.

### Gerando uma Chave Segura

Para gerar uma chave segura para produção, você pode usar:
```bash
    # Gere uma chave aleatória segura
    openssl rand -base64 64
    
    # Configure-a como variável de ambiente
    export JWT_SECRET_KEY=$(openssl rand -base64 64)
```  
    
## Notas Importantes
    
1. **Prioridade de Configuração**: A função `GetJWTSecret()` implementa uma ordem clara de prioridade: variável de ambiente específica > configuração > valor padrão.
    
2. **Segurança em Produção**: O valor padrão hardcoded deve ser usado apenas em desenvolvimento. Em produção, sempre configure um segredo único e seguro.
    
3. **Centralização**: Esta abordagem centraliza a lógica de obtenção do segredo, tornando mais fácil rastrear e modificar no futuro.
    
4. **Logs e Avisos**: Foram adicionados avisos claros quando o valor padrão inseguro está sendo usado.
    
Ao fazer essas alterações, você está removendo os valores hardcoded e implementando uma abordagem mais flexível e segura para gerenciar o segredo JWT.


Isto gerará um token JWT válido que você pode usar para autenticar requisições administrativas.

### Autenticação via API

Também é possível obter um token via API apartir do usuário admin criado anteriormente (se configurada):
```bash
    # Login para obter token JWT
    curl -X POST http://localhost:8080/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"senha123"}'
````
### Usando o Token nas Requisições

Use o token obtido nos cabeçalhos de suas requisições:
```bash
    curl -X GET http://localhost:8080/admin/apis \
      -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Gerenciando Usuários

### Obter Token de Admin

Primeiro, você precisa obter um token de autenticação:
```bash
    curl -X POST http://localhost:8080/auth/login \
      -H "Content-Type: application/json" \
      -d '{
        "username": "admin",
        "password": "sua-senha-admin"
      }'
```
### 1. Listar Usuários
```bash
    curl -X GET http://localhost:8080/admin/users \
      -H "Authorization: Bearer seu-token-aqui"
```
### 2. Criar Novo Usuário
```bash
    curl -X POST http://localhost:8080/admin/users \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "username": "novouser",
        "password": "senha123",
        "email": "novouser@exemplo.com",
        "role": "user"
      }'
```
### 3. Obter Usuário por ID
```bash
    curl -X GET http://localhost:8080/admin/users/id-do-usuario \
      -H "Authorization: Bearer seu-token-aqui"
```
### 4. Atualizar Usuário
```bash
    curl -X PUT http://localhost:8080/admin/users/id-do-usuario \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "username": "usernovo",
        "password": "novasenha123",
        "role": "editor"
      }'
```
### 5. Excluir Usuário
```bash
    curl -X DELETE http://localhost:8080/admin/users/id-do-usuario \
      -H "Authorization: Bearer seu-token-aqui"
```
## 📝 Uso da API

### Gerenciamento de Rotas

O API Gateway atua como um proxy reverso, redirecionando requisições para serviços de backend conforme a configuração de rotas.

### Cadastro de Rotas

Existem duas maneiras de cadastrar rotas:

1. Via arquivo JSON (em config/routes.json ):
```json
    [
      {
        "path": "/api/users",
        "serviceURL": "http://user-service:8000",
        "methods": ["GET", "POST", "PUT", "DELETE"],
        "headers": ["Content-Type", "Authorization"],
        "description": "Serviço de usuários",
        "isActive": true,
        "requiredHeaders": ["X-Request-ID"]
      }
    ]
```
2. Via API administrativa:
```bash
    # Registrar nova rota
    curl -X POST http://localhost:8080/admin/register \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "path": "/api/products/",
        "serviceURL": "http://product-service:8001",
        "methods": ["GET", "POST"],
        "description": "Serviço de produtos",
        "isActive": true,
        "requiredHeaders": ["X-Request-ID"]
      }'
      
      OU
      
    # Registrar nova rota com parametros
    curl -X POST http://localhost:8080/admin/register \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "path": "/api/products/:parametro",
        "serviceURL": "http://product-service:8001",
        "methods": ["GET"],
        "description": "Serviço de produtos",
        "isActive": true,
        "requiredHeaders": ["X-Request-ID"]
      }'
      
      OU
      
    # Registrar nova rota curinga
    curl -X POST http://localhost:8080/admin/register \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "path": "/api/products/*",
        "serviceURL": "http://product-service:8001",
        "methods": ["GET", "PUT"],
        "description": "Serviço de produtos",
        "isActive": true,
        "requiredHeaders": ["X-Request-ID"]
      }'  
```
### Listagem e Gerenciamento de Rotas
```bash
    # Listar todas as rotas cadastradas
    curl -X GET http://localhost:8080/admin/apis \
      -H "Authorization: Bearer seu-token-aqui"
    
    # Atualizar uma rota existente
    curl -X PUT http://localhost:8080/admin/update \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "path": "/api/products",
        "serviceURL": "http://product-service:8002",
        "methods": ["GET", "POST", "PUT"],
        "description": "API de produtos atualizada",
        "isActive": true
      }'
    
    # Excluir uma rota
    curl -X DELETE http://localhost:8080/admin/delete?path=/api/products \
      -H "Authorization: Bearer seu-token-aqui"
    
    # Diagnosticar problemas em uma rota
    curl -X GET "http://localhost:8080/admin/diagnose-route?path=/api/products" \
      -H "Authorization: Bearer seu-token-aqui"
    
    # Limpar cache de rotas (quando houver alterações que não estão sendo refletidas)
    curl -X GET http://localhost:8080/admin/clear-cache \
      -H "Authorization: Bearer seu-token-aqui"
```

### Estrutura de uma Rota
```bash
Campo             │ Descrição                           │ Obrigatório        
───────────────────┼─────────────────────────────────────┼────────────────────
path             │ Caminho da rota (ex:  /api/users )  │ Sim                
serviceURL       │ URL do serviço de backend           │ Sim                
methods          │ Métodos HTTP permitidos (array)     │ Sim                
headers          │ Cabeçalhos a serem passados (array) │ Não                
description      │ Descrição da rota                   │ Não                
isActive         │ Se a rota está ativa                │ Não (padrão: true)
requiredHeaders  │ Cabeçalhos obrigatórios             │ Não
```

## 🚦 Rate Limiting e Proteção

### Configuração Global

Configure rate limiting global no arquivo de configuração:
```yaml
    features:
      ratelimiter: true
    
    ratelimit:
      defaultLimit: 100         # Requisições por minuto por IP
      burstFactor: 1.5          # Fator de rajada
      type: "redis"             # "memory" ou "redis" 
      redisAddress: "redis:6379"
```
### Configuração por Rota

Cada rota pode ter seus próprios limites configurados durante o registro:
```bash
    curl -X POST http://localhost:8080/admin/register \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "path": "/api/sensitive",
        "serviceURL": "http://sensitive-service:8000",
        "methods": ["GET"],
        "description": "API com acesso limitado",
        "isActive": true,
        "rateLimit": {
          "requestsPerMinute": 30,
          "burstFactor": 1.2
        }
      }'
```

### Comportamento em Excesso de Requisições

Quando o limite é excedido, o API Gateway retorna:

- Status HTTP 429 (Too Many Requests)
- Cabeçalho  Retry-After  com o tempo de espera em segundos
- Corpo JSON com mensagem de erro e tempo de espera

## 🔄 Circuit Breaking

O Circuit Breaker protege os serviços de backend contra sobrecarga quando estão falhando.

### Como Funciona

1. Em condições normais, as requisições passam normalmente (circuito fechado)
2. Quando um serviço falha consistentemente, o circuito abre temporariamente
3. Durante este período, as requisições falham rapidamente sem tentar acessar o serviço
4. Após um tempo, o circuito entra em estado semiaberto, permitindo algumas requisições de teste
5. Se essas requisições de teste forem bem-sucedidas, o circuito fecha novamente

### Configuração
```yaml
    circuitbreaker:
      enabled: true
      timeout: "30s"          # Tempo de abertura do circuito
      maxRequests: 5          # Requisições permitidas no estado semiaberto
      interval: "1m"          # Intervalo para análise de falhas
      failureThreshold: 0.5   # Percentual de falhas para abrir o circuito (50%)
```

## 📊 Monitoramento e Métricas

### Métricas do Prometheus

O API Gateway expõe métricas no formato Prometheus no endpoint  /metrics :
```bash
    # Acessar métricas do Prometheus
    curl -X GET http://localhost:8080/metrics
```
Ou visualize o dashboard no Grafana em  http://localhost:3000  (usuário: admin, senha: admin por padrão).

### Principais Métricas Disponíveis

-  api_gateway_requests_total : Total de requisições por rota, método e código de status
-  api_gateway_request_duration_seconds : Duração das requisições em segundos
-  api_gateway_active_requests : Número de requisições em andamento
-  api_gateway_errors_total : Total de erros por tipo
-  api_gateway_circuit_breaker_open : Estado dos circuit breakers (1=aberto, 0=fechado)
-  api_gateway_rate_limited_requests_total : Requisições limitadas por rate limiting
-  api_gateway_cache_hit_ratio : Taxa de acerto de cache


### Visualização com Grafana

O Docker Compose inclui Grafana pré-configurado com dashboard para as métricas do API Gateway:

1. Acesse http://localhost:3000
2. Faça login (usuário: admin, senha: admin por padrão)
3. Navegue até o dashboard "API Gateway Overview"

## 🔍 Health Check e Diagnóstico

O API Gateway oferece endpoints de health check para monitoramento:
```bash
    # Verificação básica (liveness)
    curl -X GET http://localhost:8080/health
    
    # Verificação de prontidão (readiness)
    curl -X GET http://localhost:8080/health/readiness
    
    # Verificação detalhada de saúde (requer autenticação admin)
    curl -X GET http://localhost:8080/admin/health/detailed \
      -H "Authorization: Bearer seu-token-aqui"
```
### Diagnosticando Problemas

Para problemas em rotas específicas, use o endpoint de diagnóstico:
```bash
    curl -X GET "http://localhost:8080/admin/diagnose-route?path=/api/problematico" \
      -H "Authorization: Bearer seu-token-aqui"
```
Este endpoint verifica:

- Se a rota existe no banco de dados
- Se a rota está ativa
- Se a URL do serviço é válida
- Se o serviço de destino está acessível
- Latência aproximada do serviço

## 📦 Cache

O API Gateway oferece cache de resposta para melhorar a performance.

### Configuração Global
```yaml
    cache:
      enabled: true
      type: "redis"            # "memory" ou "redis"
      address: "redis:6379"    # Endereço do Redis, se aplicável
      ttl: "5m"                # Tempo de vida padrão
```
### Configuração por Rota

Cada rota pode ter suas próprias configurações de cache:
```bash
    curl -X POST http://localhost:8080/admin/register \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{
        "path": "/api/products",
        "serviceURL": "http://product-service:8000",
        "methods": ["GET"],
        "description": "Serviço de produtos",
        "isActive": true,
        "cache": {
          "enabled": true,
          "ttl": "10m"
        }
      }'
```
### Invalidação de Cache

Para invalidar o cache manualmente:
```bash
    # Limpar cache de todas as rotas
    curl -X GET http://localhost:8080/admin/clear-cache \
      -H "Authorization: Bearer seu-token-aqui"
    
    # Limpar cache de uma rota específica
    curl -X POST http://localhost:8080/admin/clear-route-cache \
      -H "Authorization: Bearer seu-token-aqui" \
      -H "Content-Type: application/json" \
      -d '{"path": "/api/products"}'
```

### Diagnóstico de Usuário

# Para PostgreSQL
```bash
go run cmd/tools/diagnose_user.go -username "admin" -driver postgres -dsn "postgres://postgres:postgres@localhost:5432/apigateway?sslmode=disable"
```    

# Para SQLite
```bash
go run cmd/tools/diagnose_user.go -username "admin" -driver sqlite -dsn "./data/apigateway.db"
```

Esta ferramenta é especialmente útil para diagnosticar problemas específicos com o armazenamento de usuários em diferentes tipos de banco de dados, permitindo comparar diretamente como os dados são armazenados e ajudando a identificar incompatibilidades.

## 🔒 Segurança Avançada

### Proteção Contra Ataques Comuns

O API Gateway implementa automaticamente várias proteções:

1. Proteção CSRF: Para rotas que exigem
2. Proteção XSS: Cabeçalhos X-XSS-Protection e Content-Security-Policy
3. Proteção contra Clickjacking: Cabeçalho X-Frame-Options
4. Proteção CORS: Controle detalhado de Cross-Origin Resource Sharing
5. Validação de Entrada: Filtragem de dados maliciosos

### Cabeçalhos de Segurança

Por padrão, o API Gateway adiciona cabeçalhos de segurança a todas as respostas:

    X-Content-Type-Options: nosniff
    X-Frame-Options: DENY
    X-XSS-Protection: 1; mode=block
    Content-Security-Policy: default-src 'self'
    Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
    Referrer-Policy: strict-origin-when-cross-origin

## 🚀 Implantação em Produção

### Checklist de Produção

Para implantação segura em produção, verifique:

[ ] Configurar chave JWT forte e armazenada com segurança
[ ] Ativar HTTPS (TLS) com certificados válidos
[ ] Configurar limites de rate limiting apropriados
[ ] Configurar banco de dados com backup automático
[ ] Ativar monitoramento e alertas
[ ] Implementar logging centralizado
[ ] Revisar todas as configurações de segurança

## 🔄 Atualizações e Migrações

### Atualizando o API Gateway

    # Atualizar o código-fonte
    git pull
    
    # Executar migrações de banco de dados
    go run cmd/migrate/main.go
    
    # Reconstruir e reiniciar o serviço
    docker-compose build api-gateway
    docker-compose up -d api-gateway

### Migrações de Banco de Dados
```bash
    # Criar nova migração
    go run cmd/migrate/main.go -action create -name add_new_field
    
    # Aplicar migrações pendentes
    go run cmd/migrate/main.go -action migrate
    
    # Reverter última migração (rollback)
    go run cmd/migrate/main.go -action rollback
```
## 📚 Arquitetura

O API Gateway foi construído seguindo os princípios de Clean Architecture:

- cmd/: Ponto de entrada da aplicação, definições de CLI
- internal/: Código específico da aplicação não reusável
- adapter/: Implementações concretas de interfaces
- app/: Casos de uso da aplicação
- domain/: Entidades de domínio e regras de negócio
- infra/: Infraestrutura como middleware e logging
- pkg/: Biblioteca reutilizável que pode ser importada por outros projetos
- config/: Arquivos de configuração
- migrations/: Migrações de banco de dados
- tests/: Testes de integração e carga

## 🤝 Contribuição

Contribuições são bem-vindas! Por favor, leia o CONTRIBUTING.md para detalhes sobre nosso código de conduta e processo de envio de Pull Requests.

## 📄 Licença

Este projeto está licenciado sob a licença MIT - veja o arquivo LICENSE para detalhes.

## ❓ Resolução de Problemas Comuns

### Token JWT Inválido ou Expirado

Se você encontrar erros com tokens JWT:

1. Verifique se o token foi gerado corretamente com  go run cmd/tools/generate_token.go
2. Certifique-se de que o segredo JWT é o mesmo no arquivo de configuração e no token
3. Verifique se o token não está expirado (duração padrão: 24h)
4. Limpe o cache do navegador caso esteja usando uma interface web

### Rotas Não Encontradas

Se suas rotas registradas não estão funcionando:

1. Verifique se a rota está registrada corretamente com  curl -X GET http://localhost:8080/admin/apis
2. Limpe o cache de rotas:  curl -X GET http://localhost:8080/admin/clear-cache
3. Verifique se o serviço de destino está acessível com  curl -X GET "http://localhost:8080/admin/diagnose-route?path=/sua/rota"
4. Verifique se o formato da rota está correto (deve começar com  /api/  ou  /ws/  por padrão)

### Problemas de Banco de Dados

Para problemas relacionados ao banco de dados:

1. Verifique as configurações de conexão no arquivo config.yaml
2. Execute  go run cmd/migrate/main.go  para aplicar migrações pendentes
3. Verifique se o banco de dados está acessível com a ferramenta adequada (psql, mysql, sqlite3)

### Segurança e Autenticação

Para problemas de autenticação:

1. Use a ferramenta de linha de comando para gerar um novo token administrativo
2. Verifique os logs para mensagens detalhadas de erro
3. Se você esqueceu a senha do administrador, crie um novo usuário admin usando a ferramenta CLI
  