# betwallet.api

Serviço de processamento de operações financeiras de provedores de jogos, com garantias de idempotência, concorrência e integridade de ledger. Ver [`ARCHITECTURE.md`](./ARCHITECTURE.md) para as decisões técnicas por trás do que está descrito aqui.

## Pré-requisitos

- Go 1.23+
- Docker e Docker Compose (v2.20+, necessário para `condition: service_completed_successfully`)
- `curl` e `jq` (para os exemplos de teste manual abaixo)

## Estrutura do projeto

```
cmd/api/            ponto de entrada da API HTTP
internal/domain/     regras de negócio puras
internal/app/        casos de uso e portas (interfaces)
internal/infra/      Postgres, HTTP/router, config, auth, geração de ids
migrations/          migrations SQL versionadas
keycloak/            realm exportado, importado automaticamente na subida
scripts/             scripts de automação (testes de integração)
```

## Subindo o ambiente

1. Clone o repositório e copie o arquivo de variáveis de ambiente:

   ```bash
   cp .env.example .env
   ```

2. Suba a infraestrutura (Postgres, LocalStack, Keycloak):

   ```bash
   docker compose up -d
   ```

   O Keycloak importa automaticamente o realm `betwallet` (clients `provider-a`, `provider-b`, `internal-service`) a partir de `keycloak/realm-export.json` — nenhum passo manual é necessário.

3. Aplique as migrations:

   ```bash
   ./scripts/migrate-run.sh up
   ```

4. Baixe as dependências Go:

   ```bash
   go mod tidy
   ```

5. Rode a aplicação:

   ```bash
   go run cmd/api/main.go
   ```

6. Confirme que subiu:

   ```bash
   curl http://localhost:8080/health/live
   curl http://localhost:8080/health/ready
   ```

## Reiniciando do zero

Se você mudar variáveis de ambiente relacionadas a usuário/senha do Postgres, ou quiser garantir um estado limpo, derrube os containers **junto com os volumes**:

```bash
docker compose down -v
docker compose up -d
```

Sem o `-v`, o Postgres reaproveita o volume de dados anterior e ignora as novas variáveis de ambiente na inicialização.

## Rodando os testes

### Testes unitários (domínio)

Rápidos, sem dependências externas:

```bash
go test ./internal/domain/... -v
```

Com detecção de race condition (requer `gcc`/CGO habilitado — em Linux, `sudo apt install build-essential`; em macOS, `xcode-select --install`):

```bash
go test ./internal/domain/... -v -race
```

### Testes de integração (casos de uso + Postgres real)

Sobem um Postgres dedicado, aplicam as migrations do zero, rodam a suíte e derrubam tudo ao final (sucesso, falha ou interrupção):

```bash
./scripts/test-integration.sh
```

Isso inclui o teste obrigatório de concorrência (seção 8 do desafio): uma carteira com 100.00 BRL recebendo duas apostas simultâneas de 80.00 BRL, validando que apenas uma é processada, a outra é rejeitada por saldo insuficiente, o saldo final é 20.00 e existe exatamente um lançamento de débito no ledger.

> **Nota:** os testes de integração chamam os casos de uso da camada de aplicação diretamente (sem passar pela camada HTTP nem pelo Keycloak). A validação de autenticação/autorização foi feita manualmente — ver seção de curls abaixo.

## Autenticação — obtendo tokens do Keycloak

Todos os endpoints de negócio exigem um token OAuth2 (`client_credentials`) emitido pelo realm `betwallet`.

```bash
# Token de um provedor de jogo (provider-a)
PROVIDER_A_TOKEN=$(curl -s -X POST http://localhost:8081/realms/betwallet/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=provider-a" \
  -d "client_secret=provider-a-secret" \
  | jq -r .access_token)

# Token de um segundo provedor, para testar isolamento
PROVIDER_B_TOKEN=$(curl -s -X POST http://localhost:8081/realms/betwallet/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=provider-b" \
  -d "client_secret=provider-b-secret" \
  | jq -r .access_token)

# Token de serviço interno (abertura de carteira, consultas administrativas)
INTERNAL_TOKEN=$(curl -s -X POST http://localhost:8081/realms/betwallet/protocol/openid-connect/token \
  -d "grant_type=client_credentials" \
  -d "client_id=internal-service" \
  -d "client_secret=internal-service-secret" \
  | jq -r .access_token)
```

| Client                     | Claim            | Usado em                                              |
| -------------------------- | ---------------- | ----------------------------------------------------- |
| `provider-a`, `provider-b` | `provider_id`    | `POST /wagering/transactions`, consultas de transação |
| `internal-service`         | `internal: true` | `POST /wallets`, consultas de carteira/ledger         |

## Exemplos de chamadas

### Abrir uma carteira (requer token interno)

```bash
curl -s -X POST http://localhost:8080/wallets \
  -H "Authorization: Bearer $INTERNAL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
    "initialBalance": { "amount": "1000.00", "currency": "BRL" }
  }' | jq
```

Resposta esperada (`201 Created`):

```json
{
  "id": "...",
  "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
  "balance": { "amount": "1000.00", "currency": "BRL" },
  "version": 1
}
```

Guarde o `id` retornado — vamos usá-lo como `WALLET_ID` nos exemplos seguintes.

```bash
WALLET_ID="cole-aqui-o-id-retornado"
```

### Processar uma aposta (BET)

```bash
curl -s -i -X POST http://localhost:8080/wagering/transactions \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" \
  -H "Idempotency-Key: provider-a:bet-001" \
  -H "Content-Type: application/json" \
  -d '{
    "externalTransactionId": "bet-001",
    "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
    "walletId": "'"$WALLET_ID"'",
    "roundId": "round-987",
    "gameId": "fortune-chimp",
    "kind": "BET",
    "money": { "amount": "25.00", "currency": "BRL" }
  }'
```

Repetir exatamente a mesma chamada (mesmo `Idempotency-Key`, mesmo corpo) deve devolver `idempotentReplay: true`, sem debitar novamente.

### Processar uma vitória (WIN)

```bash
curl -s -X POST http://localhost:8080/wagering/transactions \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" \
  -H "Idempotency-Key: provider-a:win-001" \
  -H "Content-Type: application/json" \
  -d '{
    "externalTransactionId": "win-001",
    "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
    "walletId": "'"$WALLET_ID"'",
    "roundId": "round-987",
    "gameId": "fortune-chimp",
    "kind": "WIN",
    "money": { "amount": "40.00", "currency": "BRL" }
  }' | jq
```

### Registrar uma derrota (LOSS)

```bash
curl -s -X POST http://localhost:8080/wagering/transactions \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" \
  -H "Idempotency-Key: provider-a:loss-001" \
  -H "Content-Type: application/json" \
  -d '{
    "externalTransactionId": "loss-001",
    "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
    "walletId": "'"$WALLET_ID"'",
    "roundId": "round-988",
    "gameId": "fortune-chimp",
    "kind": "LOSS",
    "money": { "amount": "0.00", "currency": "BRL" }
  }' | jq
```

### Reembolsar uma aposta (REFUND)

```bash
curl -s -X POST http://localhost:8080/wagering/transactions \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" \
  -H "Idempotency-Key: provider-a:refund-001" \
  -H "Content-Type: application/json" \
  -d '{
    "externalTransactionId": "refund-001",
    "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
    "walletId": "'"$WALLET_ID"'",
    "roundId": "round-987",
    "gameId": "fortune-chimp",
    "kind": "REFUND",
    "referenceExternalTransactionId": "bet-001",
    "money": { "amount": "25.00", "currency": "BRL" }
  }' | jq
```

### Desfazer uma operação (ROLLBACK)

```bash
curl -s -X POST http://localhost:8080/wagering/transactions \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" \
  -H "Idempotency-Key: provider-a:rollback-001" \
  -H "Content-Type: application/json" \
  -d '{
    "externalTransactionId": "rollback-001",
    "playerId": "0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1",
    "walletId": "'"$WALLET_ID"'",
    "roundId": "round-987",
    "gameId": "fortune-chimp",
    "kind": "ROLLBACK",
    "referenceExternalTransactionId": "win-001",
    "money": { "amount": "40.00", "currency": "BRL" }
  }' | jq
```

### Consultar uma transação por id

```bash
curl -s http://localhost:8080/wagering/transactions/<transactionId> \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" | jq
```

### Consultar uma transação por provedor + id externo

```bash
curl -s http://localhost:8080/providers/provider-a/wagering/transactions/bet-001 \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN" | jq
```

### Consultar saldo da carteira (requer token interno)

```bash
curl -s http://localhost:8080/wallets/$WALLET_ID \
  -H "Authorization: Bearer $INTERNAL_TOKEN" | jq
```

## Validando autenticação e isolamento entre provedores

```bash
# Sem token → 401
curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/wallets

# Token de provedor tentando abrir carteira (endpoint interno) → 403
curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/wallets \
  -H "Authorization: Bearer $PROVIDER_A_TOKEN"

# provider-b tentando ler uma transação criada por provider-a → 404
# (não 403 — o serviço não confirma nem a existência da transação para quem não é dono)
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/wagering/transactions/<transactionId-do-provider-a> \
  -H "Authorization: Bearer $PROVIDER_B_TOKEN"
```

## Variáveis de ambiente

Ver [`.env.example`](./.env.example) para a lista completa com valores de exemplo para uso local. Nenhum segredo real é necessário para rodar o ambiente localmente — os valores padrão já funcionam com o `docker-compose.yml` fornecido.

## Limitações conhecidas

Ver a seção 11 de [`ARCHITECTURE.md`](./ARCHITECTURE.md) para a lista completa do que não foi implementado nesta entrega (consumidor SQS, worker de publicação da outbox, fluxo assíncrono de referência pendente, entre outros).
