# Leilão (Auction) - Full Cycle Challenge

Sistema de leilão desenvolvido em Go com fechamento automático de leilões baseado em tempo.

## 📋 Funcionalidades

- **Criação de Leilões**: Criação de leilões com duração configurável
- **Fechamento Automático**: Goroutine que fecha automaticamente o leilão após o tempo definido
- **Sistema de Lances (Bids)**: Validação automática se o leilão está ativo antes de aceitar lances
- **API REST**: Interface HTTP para todas as operações

## 🏗️ Arquitetura

O projeto segue uma arquitetura limpa (Clean Architecture) com as seguintes camadas:

```
├── cmd/auction/          # Ponto de entrada da aplicação
├── configuration/        # Configurações (database, logger)
├── internal/
│   ├── entity/           # Entidades do domínio
│   ├── infra/
│   │   ├── api/web/      # Controllers HTTP
│   │   └── database/     # Repositórios MongoDB
│   ├── internal_error/   # Tratamento de erros
│   └── usecase/          # Casos de uso
```

## 🔧 Implementação do Fechamento Automático

### Função de Cálculo de Tempo

A função `getAuctionDuration()` em `internal/infra/database/auction/create_auction.go` calcula a duração do leilão baseada na variável de ambiente `AUCTION_DURATION_SECONDS`:

```go
func getAuctionDuration() time.Duration {
    v := os.Getenv("AUCTION_DURATION_SECONDS")
    if v == "" {
        return time.Duration(600) * time.Second // Default: 10 minutos
    }
    secs, err := strconv.Atoi(v)
    if err != nil || secs <= 0 {
        return time.Duration(600) * time.Second
    }
    return time.Duration(secs) * time.Second
}
```

### Goroutine de Fechamento Automático

Quando um leilão é criado, uma goroutine é disparada para fechá-lo automaticamente:

```go
go func(auctionID string, wait time.Duration) {
    if wait > 0 {
        timer := time.NewTimer(wait)
        <-timer.C
    }

    // Atualiza o status do leilão para Finished
    filter := bson.M{"_id": auctionID, "status": Active}
    update := bson.M{"$set": bson.M{"status": Finished}}
    
    // Executa a atualização no MongoDB
    ar.Collection.FindOneAndUpdate(ctx, filter, update, opts)
}(auctionEntityMongo.Id, remaining)
```

### Tratamento de Concorrência

A solução utiliza:
- **MongoDB FindOneAndUpdate**: Operação atômica que garante que apenas uma goroutine feche o leilão
- **Context com Timeout**: Previne operações bloqueadas indefinidamente
- **Verificação de Status**: O filtro `status: Active` garante que leilões já fechados não sejam processados

## 🚀 Como Executar

### Pré-requisitos

- Docker
- Docker Compose

### Configuração das Variáveis de Ambiente

O arquivo `.env` em `cmd/auction/.env` contém as configurações:

```env
# MongoDB Configuration
MONGODB_URL=mongodb://mongodb:27017
MONGODB_DB=auctions

# Auction Configuration
AUCTION_DURATION_SECONDS=60    # Duração do leilão em segundos
AUCTION_INTERVAL=5m            # Intervalo para validação de bids
```

### Executando com Docker Compose

1. **Clone o repositório** (se ainda não fez):
```bash
git clone <repository-url>
cd FC-Action
```

2. **Crie o arquivo .env** (se não existir):
```bash
cat > cmd/auction/.env << 'EOF'
MONGODB_URL=mongodb://mongodb:27017
MONGODB_DB=auctions
AUCTION_DURATION_SECONDS=60
AUCTION_INTERVAL=5m
EOF
```

3. **Inicie os containers**:
```bash
docker-compose up --build
```

4. **A API estará disponível em**: http://localhost:8080

### Parando a aplicação

```bash
docker-compose down
```

Para remover também os volumes (dados do MongoDB):
```bash
docker-compose down -v
```

## 📡 Endpoints da API

### Leilões (Auctions)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/auction` | Lista todos os leilões |
| GET | `/auction/:auctionId` | Busca leilão por ID |
| POST | `/auction` | Cria novo leilão |
| GET | `/auction/winner/:auctionId` | Busca lance vencedor |

### Lances (Bids)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/bid` | Cria novo lance |
| GET | `/bid/:auctionId` | Lista lances de um leilão |

### Usuários (Users)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/user/:userId` | Busca usuário por ID |

## 📝 Exemplos de Requisições

### Criar um Leilão

```bash
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "iPhone 15 Pro",
    "category": "Electronics",
    "description": "iPhone 15 Pro 256GB em perfeito estado",
    "condition": 1
  }'
```

**Condições disponíveis:**
- `1`: Novo
- `2`: Usado
- `3`: Recondicionado

### Listar Leilões

```bash
curl http://localhost:8080/auction
```

### Criar um Lance

```bash
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "auction_id": "<auction_id>",
    "amount": 1500.00
  }'
```

## 🧪 Executando os Testes

### Testes Locais (requer MongoDB rodando)

```bash
# Inicie apenas o MongoDB
docker-compose up mongodb -d

# Execute os testes
go test ./internal/infra/database/auction/... -v

# Ou execute todos os testes
go test ./... -v
```

### Testes dentro do Docker

```bash
# Construa a imagem de teste
docker-compose exec app go test ./internal/infra/database/auction/... -v
```

## 📁 Estrutura de Arquivos Principais

```
├── cmd/auction/
│   ├── main.go              # Ponto de entrada
│   └── .env                 # Variáveis de ambiente
├── internal/
│   └── infra/database/auction/
│       ├── create_auction.go       # Implementação com goroutine
│       ├── create_auction_test.go  # Testes automatizados
│       └── find_auction.go         # Busca de leilões
├── docker-compose.yml
├── Dockerfile
└── README.md
```

## 🔍 Verificando o Fechamento Automático

Para verificar o funcionamento do fechamento automático:

1. Configure `AUCTION_DURATION_SECONDS=30` para 30 segundos
2. Crie um leilão
3. Aguarde 30 segundos
4. Busque o leilão novamente - o status deve ser `1` (Completed)

```bash
# Criar leilão
AUCTION_ID=$(curl -s -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "Test Product",
    "category": "Test",
    "description": "Testing auto-close feature",
    "condition": 1
  }' | jq -r '.id')

echo "Leilão criado: $AUCTION_ID"

# Verificar status (deve ser 0 = Active)
curl http://localhost:8080/auction/$AUCTION_ID

# Aguardar o tempo configurado + margem
sleep 35

# Verificar status novamente (deve ser 1 = Completed)
curl http://localhost:8080/auction/$AUCTION_ID
```

## 📊 Status dos Leilões

| Código | Status | Descrição |
|--------|--------|-----------|
| 0 | Active | Leilão aberto para lances |
| 1 | Completed | Leilão fechado automaticamente |

## 🛠️ Tecnologias Utilizadas

- **Go 1.20**: Linguagem principal
- **Gin**: Framework web
- **MongoDB**: Banco de dados
- **Docker/Docker Compose**: Containerização
- **Zap**: Logger estruturado

## 📄 Licença

Este projeto foi desenvolvido como parte do desafio Full Cycle.
