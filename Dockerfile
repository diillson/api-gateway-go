# syntax=docker/dockerfile:1

# Estágio de compilação
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Instalar dependências de compilação
RUN apk --no-cache add ca-certificates git tzdata

# Baixar dependências primeiro (para melhor aproveitar o cache)
COPY go.mod go.sum ./
RUN go mod download

# Copiar o código fonte
COPY . .

# Compilar a aplicação
RUN GOOS=linux GOARCH=amd64 go build -a -ldflags="-w -s -X main.version=$(git describe --tags --always)" -o apigateway ./cmd/apigateway

# Estágio final - imagem mínima
FROM scratch

WORKDIR /app

    # Copiar arquivos necessários da etapa de compilação
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/apigateway /app/apigateway
COPY --from=builder /app/config /app/config
COPY --from=builder /app/migrations /app/migrations

    # Definir variáveis de ambiente padrão
ENV AG_SERVER_PORT=8080 \
    AG_LOGGING_LEVEL=info \
    AG_LOGGING_FORMAT=json \
    AG_LOGGING_PRODUCTION=true

    # Expor porta
EXPOSE 8080

    # Definir entrypoint
ENTRYPOINT ["/app/apigateway"]
CMD ["server --config /app/config/config.yaml"]