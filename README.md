# API Gateway em Go

## 🚀 Visão Geral

Este projeto é um API Gateway robusto, escalável e eficiente, escrito em Go. Ele atua como um ponto intermediário que gerencia e manipula solicitações de API de clientes para serviços de backend. Equipado com recursos como autenticação, limitação de taxa, logging e análise, este Gateway é a solução perfeita para gerenciar, otimizar e proteger suas APIs.

## 📦 Recursos

- **Autenticação JWT:** Secure suas APIs com a autenticação baseada em Token JWT.
- **Limitação de Taxa:** Protege seus serviços de backend de serem sobrecarregados por muitas requisições.
- **Logging e Análise:** Mantenha um olho no desempenho e na utilização de suas APIs com registros detalhados e análises.
- **Modular e Extensível:** O código é escrito de maneira modular e pode ser facilmente estendido ou modificado.


### Bibliotecas Utilizadas no API Gateway Go

Neste projeto de API Gateway, várias bibliotecas Go poderosas e eficientes são utilizadas para garantir a funcionalidade, escalabilidade e desempenho otimizados. Aqui está um olhar mais detalhado sobre cada uma delas:

#### 1. **Gin**
- **Website:** [Gin Web Framework](https://gin-gonic.com/)
- **Descrição:** Gin é um framework web HTTP para construir APIs. Ele é conhecido por sua velocidade e baixo consumo de memória, tornando-se uma escolha popular para aplicativos que necessitam de desempenho otimizado. No API Gateway, Gin é usado para manipular solicitações HTTP, rotas e middleware.

#### 2. **Gorm**
- **Website:** [GORM](https://gorm.io/)
- **Descrição:** GORM é um ORM (Object-Relational Mapping) para Go. Ele ajuda na manipulação de bancos de dados, oferecendo uma interface amigável para realizar operações como Create, Read, Update e Delete (CRUD). Neste projeto, GORM é utilizado para gerenciar e operar no banco de dados SQLite.

#### 3. **Zap**
- **Website:** [Zap](https://go.uber.org/zap)
- **Descrição:** Zap é uma biblioteca de logging para Go. É rápida e oferece uma interface flexível para registrar mensagens em vários níveis de severidade. Neste projeto, Zap é empregado para capturar, registrar e monitorar as atividades e operações do API Gateway.

#### 4. **JWT-Go**
- **GitHub:** [JWT-Go](https://github.com/golang-jwt/jwt)
- **Descrição:** JWT-Go é uma biblioteca Go para criar e validar tokens JWT (JSON Web Tokens). É eficiente e fácil de usar. No contexto deste API Gateway, JWT-Go é utilizado para implementar a autenticação baseada em tokens.

#### 5. **Rate**
- **Parte do pacote:** [x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- **Descrição:** Esta biblioteca é parte do pacote x/time do Go e é usada para implementar a limitação de taxa. No projeto, é aplicada para controlar o número de solicitações que um usuário pode fazer em um período específico, prevenindo assim abusos e garantindo a qualidade do serviço.

### Considerações

Estas bibliotecas foram escolhidas pela sua eficiência, facilidade de uso e comunidade ativa. Elas se integram perfeitamente para criar um API Gateway robusto e eficiente. O Gin oferece velocidade e eficiência, o GORM oferece uma manipulação de banco de dados simplificada, o Zap garante que todas as atividades sejam registradas e monitoradas de forma eficaz, e o JWT-Go assegura que a autenticação e a segurança estejam no seu melhor.

Ao utilizar estas bibliotecas juntas, conseguimos criar um sistema que não só é performático e seguro, mas também fácil de manter e expandir, garantindo assim que o API Gateway possa escalar e evoluir junto com as necessidades do negócio.


## 🛠️ Instalação e Configuração

1. **Clone o Repositório:**
    ```sh
    git clone https://github.com/diillson/api-gateway-go.git
    cd api-gateway-go
    ```

2. **Instale as Dependências:**
    ```sh
    go mod tidy
    ```

3. **Inicialize o Servidor:**
    ```sh
   cd cmd 
   go run main.go
    ```
Agora o ApiGateway estará rodando no `http://localhost:8080`. Você receberá um token JWT no console após iniciar o servidor.
Perceba caso desejar já iniciar o servidor com apis cadastradas, basta adicionar no routes.json dentro da pasta raiz de seu projeto conforme a estrutura "./routes/routes.json"

# **Build**

### MacOS
    #amd64
    GOOS=darwin GOARCH=amd64 go build -o cmd/apigateway cmd/main.go

    #arm64
    GOOS=darwin GOARCH=arm64 go build -o cmd/apigateway cmd/main.go

### Linux

    # amd64
    $ GOOS=linux GOARCH=amd64 go build -o cmd/apigateway cmd/main.go

    # arm64
    $ GOOS=linux GOARCH=arm64 go build -o cmd/apigateway cmd/main.go

### Windows

    # amd64
    $ GOOS=windows GOARCH=amd64 go build -o cmd/apigateway.exe cmd/main.go
    
    # arm64
    $ GOOS=windows GOARCH=arm64 go build -o cmd/apigateway.exe cmd/main.go

## 📚 Uso

Para autenticar e acessar as rotas protegidas, você precisará usar o token JWT gerado. O Gateway oferece endpoints para listar, adicionar, atualizar e deletar rotas, bem como para visualizar métricas.
passando o seguinte Headers nas request:

    Header: Authorization
    Value: Bearer seu-token

- **Autenticar:**
    - Use o JWT token para fazer requisições autorizadas aos endpoints protegidos.

- **Adicionar Rotas:**
    - Faça uma requisição POST para `/admin/register` com os detalhes da rota no corpo para adicionar novas rotas.

- **Visualizar Rotas:**
    - Faça uma requisição GET para `/admin/apis` para ver todas as rotas registradas.

- **Atualizar Rotas:**
    - Faça uma requisição PUT para `/admin/update` com os novos detalhes da rota para atualizá-la.

- **Deletar Rotas:**
    - Faça uma requisição DELETE para `/admin/delete` com o caminho da rota na query para deletá-la.

- **Visualizar Métricas:**
    - Faça uma requisição GET para `/admin/metrics` para visualizar métricas.

## 🛡️ Segurança

O projeto utiliza autenticação JWT para garantir que apenas usuários autorizados possam acessar os endpoints administrativos. Além disso, a limitação de taxa está em vigor para prevenir abusos e garantir a disponibilidade do serviço.

## 👩‍💻 Contribuição

Sinta-se à vontade para abrir issues ou pull requests se você deseja melhorar ou discutir algo sobre o projeto.

## 📄 Licença

Este projeto está sob a licença GPL - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🌟 Agradecimentos

Agradecemos a todos que de alguma forma poder contribuir e apoiar o desenvolvimento deste projeto. Sua ajuda é inestimável para tornar este projeto ótimo!