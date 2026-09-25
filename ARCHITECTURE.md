# ARCHITECTURE.md — betwallet.api

Este documento registra as decisões técnicas tomadas na construção do serviço, os motivos por trás delas, e o que ficou fora do escopo desta entrega.

---

## 1. Visão geral

O serviço expõe uma API HTTP e (planejado, não implementado) um consumidor SQS para movimentar carteiras de jogadores a partir de operações enviadas por provedores de jogos externos. A composição da aplicação é feita com Uber Fx; a persistência com PostgreSQL via `pgx/v5`, com SQL explícito (sem ORM); autenticação via OAuth2/OIDC (Keycloak).

O domínio (`internal/domain`) é isolado de qualquer dependência de infraestrutura — não importa Fx, `net/http`, `database/sql`/`pgx`, nem SDKs de mensageria. A camada de aplicação (`internal/app`) orquestra o domínio através de interfaces (portas) que ela mesma declara; a infraestrutura (`internal/infra/*`) implementa essas portas.

```
cmd/api/            ponto de entrada — só compõe módulos do Fx
internal/domain/     regras de negócio puras (Money, Wallet, WagerTransaction, ...)
internal/app/        casos de uso, portas (interfaces), envelope de eventos
internal/infra/
  config/            leitura de variáveis de ambiente
  http/, router/      servidor HTTP e handlers
  postgres/           implementação dos repositórios e da unit of work
  idgen/              geração de identificadores (UUID v7)
  auth/               validação de token OIDC / middlewares de autorização
migrations/           migrations SQL versionadas
```

---

## 2. Dinheiro (Money)

- Representação interna em **unidades mínimas (centavos)**, tipo `int64` — nunca `float32`/`float64`, em nenhum ponto do parsing, cálculo, serialização ou persistência.
- `Money` é um **value object imutável**: toda operação (`Add`, `Sub`, `Negate`) devolve um novo valor; nenhum método muta o receiver.
- O construtor `Parse(amount, currency)` aceita **apenas o formato canônico** `-?\d+\.\d{2}` (duas casas decimais exatas). Não há normalização de formas equivalentes (`"25"`, `"25.0"`, `"25.000"` são todas rejeitadas, nunca corrigidas silenciosamente) — isso mantém o hash de idempotência determinístico sem depender de regras de normalização implícitas.
- Overflow é tratado explicitamente em `Parse`, `Add` e `Negate` (o caso de `math.MinInt64` não ter um positivo correspondente é tratado como erro).
- Sinal negativo é permitido no tipo `Money` em si (necessário para diferenças e cálculos internos), mas **quem decide** se um valor negativo é aceitável em determinado contexto é a camada que o consome (`Wallet`, `WagerTransaction`) — o tipo `Money` não assume que só representa saldos.
- Persistência: coluna `BIGINT` em unidades mínimas, junto de uma coluna de moeda (`CHAR(3)`, ISO 4217).
- Contrato externo: `{"amount":"25.00","currency":"BRL"}`, via `MarshalJSON`/`UnmarshalJSON` customizados no tipo `Money`.
- Suporte a múltiplas moedas: `Money` carrega sua moeda; operações entre moedas incompatíveis (`Add`, `Sub`, `Cmp`) retornam erro (`ErrCurrencyMismatch`). Os cenários principais e os testes de integração usam apenas BRL, mas o tipo não está restrito a uma única moeda.

---

## 3. Transações (WagerTransaction) e máquina de estados

- Estados: `PENDING → PROCESSED | REJECTED | FAILED`, com um estado intermediário `PENDING_REFERENCE` para REFUND/ROLLBACK cuja referência ainda não chegou.
- As transições são métodos do próprio tipo (`MarkProcessed`, `MarkRejected`, `MarkFailed`, `MarkPendingReference`), que validam o estado atual antes de mudar. Um estado terminal (`PROCESSED`, `REJECTED`, `FAILED`) nunca sofre nova transição — isso é garantido no domínio, não apenas por convenção de uso.
- Falha transitória de infraestrutura (`FAILED`) é distinta de rejeição de negócio (`REJECTED`): a primeira é auditoria de um problema de infraestrutura; a segunda é um resultado de negócio válido e esperado (ex: saldo insuficiente).
- `OPENING` é um tipo reservado à abertura interna de carteira. O construtor `NewExternalWagerTransaction` rejeita `OPENING` explicitamente; apenas `NewOpeningTransaction` pode criá-lo, e esse construtor nunca é alcançável a partir de um handler HTTP ou do consumidor SQS.
- O schema distingue operação interna de externa via uma `CHECK` constraint: uma linha `OPENING` nunca tem os metadados externos preenchidos (provider, external id, idempotency key, payload hash, round, game); uma linha externa sempre os tem.
- Separação entre **criação** (`New*`) e **reidratação** (`Rehydrate*`): a reidratação reconstrói o estado tal como persistido, sem reaplicar nenhuma regra de negócio, transição ou emissão de evento.

### 3.1. Regras de valor por tipo de operação

| Tipo     | Movimento                     | Regra de valor                                        |
| -------- | ----------------------------- | ----------------------------------------------------- |
| BET      | Débito                        | `money > 0`                                           |
| WIN      | Crédito                       | `money > 0`                                           |
| LOSS     | Nenhum                        | `money == 0.00` exatamente                            |
| REFUND   | Crédito                       | `money > 0`, igual ao valor da BET referenciada       |
| ROLLBACK | Inverso da transação original | `money > 0`, igual ao valor da transação referenciada |

LOSS nunca gera lançamento de ledger nem altera a versão da carteira — apenas o evento `WagerTransactionProcessed` é publicado.

---

## 4. Idempotência

- O header `Idempotency-Key` é obrigatório em `POST /wagering/transactions`. O servidor **não** substitui a chave recebida por uma calculada — o cliente é responsável por sua construção (`{providerId}:{externalTransactionId}` é a convenção sugerida, não imposta).
- Um **hash canônico determinístico** dos campos de negócio é calculado (SHA-256 sobre um JSON com um conjunto fixo de campos, sempre na mesma ordem de declaração do struct): `externalTransactionId`, `gameId`, `kind`, `money.amount`, `money.currency`, `playerId`, `providerId`, `referenceExternalTransactionId` (quando aplicável), `roundId`, `walletId`.
  - **Excluídos do hash, propositalmente**: a própria idempotency key e qualquer metadado de transporte (headers HTTP, `messageId` do SQS, timestamps de recebimento).
  - Não há normalização de formas equivalentes antes do hash — os campos já chegam no formato canônico exigido pelo domínio (ex: `Money.String()` sempre produz duas casas decimais).
- Fluxo de checagem, no início do caso de uso, **antes** de qualquer tentativa de escrita:
  - Chave conhecida + hash igual → devolve o resultado persistido (`idempotentReplay: true`), sem reprocessar.
  - Chave conhecida + hash diferente → `ErrIdempotencyConflict` (mapeado para `409 Conflict` no HTTP).
  - Chave desconhecida → segue para o processamento normal.
- **Corrida entre duas requisições idênticas simultâneas**: a checagem inicial por si só não cobre o caso de duas requisições chegarem exatamente ao mesmo tempo, ambas sem encontrar registro ainda. Essa corrida é resolvida por uma `UNIQUE INDEX (provider_id, external_transaction_id)` no banco — quando o `INSERT` da transação perdedora viola essa constraint, o erro é mapeado para `ErrIdempotencyKeyRaceLost`, e o caso de uso então busca e devolve o resultado da transação vencedora como replay, em vez de propagar um erro de infraestrutura.
- Uma operação identificada por `(providerId, externalTransactionId)` não pode ser reaplicada usando outra idempotency key — a unicidade no banco é sobre o par `(providerId, externalTransactionId)`, não sobre a chave em si.
- O mesmo cálculo de hash e a mesma lógica de checagem valem tanto para entrada via HTTP quanto (quando implementado) via SQS — a diferença é apenas de onde a idempotency key é extraída (header HTTP vs. `data.idempotencyKey` no corpo da mensagem).

---

## 5. Concorrência

- **Estratégia escolhida: controle otimista via coluna `version`**, sem locks explícitos (`SELECT ... FOR UPDATE`) e sem fila de espera em memória.
- Mecanismo: o caso de uso lê a carteira (e sua `version` atual), aplica a mutação em memória, e tenta `UPDATE wallets SET balance_minor_units = $1, version = $2, updated_at = $3 WHERE id = $4 AND version = $5` — o `$5` é a versão lida antes da mutação. Se `RowsAffected() == 0`, outro processo já avançou a versão primeiro; o caso de uso trata isso como `ErrConcurrentUpdate` e tenta novamente **do zero**, em uma **nova transação SQL**, até um limite de tentativas (5).
- A garantia central mora inteiramente no banco (a condição `WHERE version = $x` no `UPDATE`) — não em nenhuma coordenação em memória do processo Go. Isso é o que permite múltiplas instâncias da aplicação, sem estado compartilhado, operarem sobre a mesma carteira sem corromper o saldo.
- Carteiras diferentes nunca se bloqueiam entre si: não existe lock global; cada tentativa de `UPDATE` afeta apenas a linha da carteira envolvida.
- Cada tentativa de retry roda em sua própria transação (não se mantém uma transação aberta enquanto se espera ou se tenta de novo).
- O mesmo mecanismo resolve, sem lógica adicional, a regra "uma referência não pode receber duas reversões bem-sucedidas do mesmo tipo": uma `UNIQUE INDEX (resolved_reference_id) WHERE status = 'PROCESSED' AND kind IN ('REFUND', 'ROLLBACK')` faz o Postgres arbitrar a corrida entre duas reversões concorrentes sobre a mesma referência; a violação é mapeada para `ErrReversalRaceLost` e tratada como nova tentativa.
- Validado por teste de integração (`TestIntegration_ConcurrentBets_OnlyOneSucceeds`): duas goroutines, cada uma com sua própria transação/conexão, disparando BET de 80.00 simultaneamente sobre uma carteira de 100.00. Resultado: uma `PROCESSED`, uma `REJECTED` por saldo insuficiente, saldo final 20.00, exatamente um lançamento de débito no ledger.

### 5.1. Limitação conhecida

O teste de concorrência usa goroutines dentro do mesmo processo `go test`, cada uma com sua própria conexão de pool — isso prova a garantia de concorrência no nível do banco (que é onde ela realmente é imposta), mas não é literalmente "três processos do sistema operacional" como sugerido na seção 8 do desafio. Não foi implementado, por restrição de tempo, um CLI auxiliar para disparar três processos SO independentes.

---

## 6. Referências e reversões (REFUND/ROLLBACK)

- Resolução da referência: por `(providerId, referenceExternalTransactionId)`, via `FindByProviderAndExternalID`.
- Validações aplicadas, na ordem, antes de processar uma reversão:
  1. Referência existe (`FailureReferenceNotFound` se não).
  2. Para REFUND: a referência precisa ser do tipo `BET` (`FailureReferenceKindInvalid`).
  3. A referência precisa estar em estado `PROCESSED` (`FailureReferenceNotProcessed`).
  4. Jogador, carteira, rodada e moeda da reversão precisam concordar com os da referência (`FailureReferenceMismatch`).
  5. O valor da reversão precisa ser exatamente igual ao valor da referência — reversões parciais não são suportadas (`FailureReferenceMismatch`).
  6. A referência ainda não pode ter sido revertida com sucesso por nenhum REFUND ou ROLLBACK anterior (`FailureReferenceAlreadyReversed`), verificado tanto por checagem antecipada quanto pela `UNIQUE INDEX` no banco (ver seção 5).
- Código de falha diferenciado para saldo insuficiente: uma **aposta** sem saldo usa `INSUFFICIENT_BALANCE`; uma **reversão** que debitaria mais que o saldo disponível usa `REVERSAL_INSUFFICIENT_BALANCE` — são cenários de negócio distintos e o desafio exige que sejam distinguíveis.
- `ROLLBACK` inverte o movimento da transação original: se a original foi um débito (BET), o rollback credita; se foi um crédito (WIN ou REFUND), o rollback debita.

### 6.1. Limitação conhecida — PENDING_REFERENCE não implementado

O fluxo assíncrono de referência pendente (persistir como `PENDING_REFERENCE`, retry com backoff exponencial via worker dedicado, expiração por número máximo de tentativas ou TTL resultando em `REJECTED`) **não foi implementado** nesta entrega. Hoje, se a referência de um REFUND/ROLLBACK não é encontrada, a operação é rejeitada imediatamente com `FailureReferenceNotFound`, em vez de aguardar a chegada posterior da referência. Isso é uma simplificação consciente por restrição de tempo — a estrutura de estados (`PENDING_REFERENCE` já existe no domínio) suportaria essa extensão sem mudança de schema.

### 6.2. Limitação conhecida — referência informativa de WIN não persistida

O enunciado permite que um `WIN` informe, opcionalmente, uma `BET` da mesma rodada como referência. Essa referência não é persistida nem usada para nenhuma validação nesta entrega — `WIN` é tratado como um crédito simples, sem vínculo de referência gravado.

---

## 7. Ledger

- Tabela `wallet_ledger_entries`, com colunas `direction` (`DEBIT`/`CREDIT`), `amount`, `balance_before`, `balance_after` (todas em unidades mínimas), e `(wallet_id, transaction_id)` com `UNIQUE` constraint.
- **Append-only imposto no banco**, não apenas por convenção de código: triggers `BEFORE UPDATE` e `BEFORE DELETE` na tabela levantam exceção, impedindo qualquer edição ou remoção de um lançamento já gravado.
- A construção de um `WalletLedgerEntry` (`NewWalletLedgerEntry`) **recalcula** `balanceAfter = balanceBefore ± amount` (conforme a direção) e recusa a criação se o valor informado divergir do calculado — essa validação roda tanto na criação quanto na reidratação (`RehydrateWalletLedgerEntry` delega para o mesmo construtor), como uma segunda camada de proteção independente do que o `Wallet.Debit`/`Credit` já garante.
- `LOSS` e operações rejeitadas nunca produzem lançamento de ledger.
- Reconciliação (`POST /wallets/:walletId/reconciliation`) reconstrói o saldo a partir da soma de créditos menos débitos do ledger (incluindo a abertura) e compara com o saldo armazenado.
- Ledger de partidas dobradas (double-entry) não foi implementado — é citado no desafio como diferencial opcional.

---

## 8. Inbox e Outbox

- **Outbox**: implementado e em uso. Toda publicação de evento (`WagerTransactionProcessed`, `WagerTransactionRejected`, `WalletBalanceChanged`) é persistida em `outbox_entries` **na mesma transação SQL** que confirma a mudança de domínio correspondente — nunca há publicação direta em um broker a partir do caso de uso. Isso garante que um evento só existe no outbox se a operação que o originou de fato foi commitada.
  - Campos: `event_id` (estável através de republicações), `aggregate_id`, `event_type`, `payload` (snapshot JSON imutável, serializado no momento da criação), `occurred_at`, `attempts`, `next_attempt_at`, `published_at` (nulo até a publicação ser confirmada).
  - `ScheduleRetry(backoff)` incrementa `attempts` e empurra `next_attempt_at`; a fórmula de backoff em si é decisão de quem chama, não do domínio.
- **Inbox**: o tipo de domínio (`InboxEntry`) foi modelado, mas **não está em uso**, porque o consumidor SQS não foi implementado nesta entrega (ver seção 11).
- **Worker de publicação da outbox**: não implementado. Os eventos são gravados corretamente na tabela, mas nenhum processo os lê e publica de fato em um broker/tópico.

### 8.1. Limitação conhecida

A garantia "evento só é publicado depois do commit" está corretamente implementada do lado da escrita (outbox gravado na mesma transação). A leitura/publicação de fato (worker separado, com suporte a múltiplos publishers concorrentes, backoff e recuperação de trabalho abandonado) **não foi implementada** por restrição de tempo.

---

## 9. Autenticação e autorização

- IdP: **Keycloak**, escolhido por ser a opção recomendada no desafio e por suportar `client_credentials` nativamente, sem exigir implementação própria de emissão de token.
- Realm (`betwallet`) e clients provisionados automaticamente na subida via _realm import_ do Keycloak (`--import-realm` + arquivo `keycloak/realm-export.json` montado como volume) — sem passo manual ou chamada à API administrativa do Keycloak.
- Três clients de exemplo, todos usando `client_credentials`:
  - `provider-a`, `provider-b`: carregam a claim `provider_id` (via _hardcoded claim mapper_), representando provedores de jogo externos.
  - `internal-service`: carrega a claim `internal: true`, representando chamadas internas (abertura de carteira).
- Validação do token: `github.com/coreos/go-oidc/v3`, via discovery automático do documento OIDC do issuer (`/.well-known/openid-configuration`) na subida da aplicação — se o Keycloak não estiver acessível nesse momento, a aplicação falha ao iniciar, com erro explícito, em vez de aceitar requisições sem conseguir validar tokens.
- Modelo de autorização, aplicado via middlewares HTTP compostos por handler:
  - `RequireAuth`: valida assinatura e expiração do token; extrai `provider_id`/`internal` para o contexto da requisição.
  - `RequireProvider`: exige que o token carregue `provider_id` — aplicado a `POST /wagering/transactions` e às consultas de transação por provedor.
  - `RequireInternal`: exige `internal: true` — aplicado a `POST /wallets` e às consultas de carteira/ledger, que são tratadas como operações administrativas/internas, nunca expostas diretamente a um provedor de jogo.
- **A identidade do provedor nunca é confiada a partir do corpo da requisição.** Em `POST /wagering/transactions`, o `providerId` usado no processamento vem exclusivamente da claim do token autenticado — o campo `providerId` que porventura apareça no corpo é ignorado para fins de autorização, evitando que um provedor se passe por outro editando o payload.
- Isolamento entre provedores em consultas: a busca de uma transação por id filtra por `provider_id` diretamente na cláusula `WHERE` da query (`FindByID(transactionID, providerID)`), de modo que uma transação de outro provedor resulta em "não encontrado" (`404`), nunca em um vazamento de existência ou dados de outro provedor.

### 9.1. Limitação conhecida

Não existe suíte de teste automatizada para a camada de autenticação/autorização — a validação foi feita manualmente (Postman/curl), incluindo o cenário de isolamento entre `provider-a` e `provider-b`. Os testes de integração automatizados (`-tags=integration`) exercitam os casos de uso da camada de aplicação diretamente contra Postgres real, sem passar pela camada HTTP nem pelo Keycloak.

---

## 10. Composição com Uber Fx

- Cada pacote de infraestrutura expõe seu próprio `fx.Module`, agrupando `fx.Provide`/`fx.Invoke` relacionados; `cmd/api/main.go` apenas lista os módulos, sem conhecer detalhes de construção de nenhum deles.
- **Value groups** (`group:"routes"`) são usados para o registro de handlers HTTP: cada handler implementa uma interface `Route` e se registra no grupo via `fx.Annotate(..., fx.As(new(Route)), fx.ResultTags(\`group:"routes"\`))`; o router consome `[]Route`via um struct com`fx.In`. Isso permite adicionar um novo endpoint sem tocar no código do router — só o `Module`do pacote`router` cresce uma linha.
- `fx.Lifecycle` gerencia o ciclo de vida do servidor HTTP e do pool de conexões Postgres:
  - Servidor HTTP: `OnStart` inicia o listener em uma goroutine (não bloqueia o `fx.New(...).Run()`); `OnStop` chama `Shutdown` com timeout de 10s, interrompendo novas conexões e aguardando as em andamento.
  - Pool Postgres: `OnStart` executa um `Ping` com timeout, validando conectividade antes de a aplicação ser considerada "de pé" (falha cedo, com erro claro, se o banco não estiver acessível); `OnStop` fecha o pool.
- Ordem de desligamento: o Fx dispara `OnStop` na ordem reversa da construção — como o servidor HTTP depende (transitivamente) do pool Postgres, o servidor é desligado antes do pool ser fechado, evitando que requisições em andamento percam a conexão com o banco no meio do processamento.

---

## 11. O que não foi implementado (escopo não concluído)

Listado aqui de forma explícita, conforme pedido no desafio:

- **Consumidor SQS** e as filas `wager-transactions.fifo` / `wager-transactions-dlq.fifo` — não implementados. `LocalStack` está provisionado no `docker-compose.yml`, mas nenhum código de aplicação o utiliza ainda.
- **Worker de publicação da outbox** — os eventos são gravados corretamente na tabela `outbox_entries` (ver seção 8), mas não há processo que os leia e publique de fato.
- **Fluxo assíncrono de `PENDING_REFERENCE`** com retry/backoff via worker — ver seção 6.1.
- **Referência informativa de WIN** — ver seção 6.2.
- **CLI de teste de concorrência com múltiplos processos SO** — ver seção 5.1.
- **Testes automatizados de autenticação/autorização via HTTP** — ver seção 9.1.
- **Tracing (OpenTelemetry)**, **dashboards** e **testes de carga** — citados no desafio como diferenciais opcionais; não implementados.
- **Ledger de partidas dobradas** — citado como diferencial opcional; não implementado.
- **Métricas** (resultados por status, duplicatas, retries, DLQ, atraso de outbox, latência, divergências de reconciliação) — não implementadas; os health checks (`/health/live`, `/health/ready`) e logs estruturados (JSON, via `log/slog`) existem, mas não há exposição de métricas (ex: Prometheus).

## 12. Interpretações adotadas

- O par `(playerId, currency)` identifica unicamente uma carteira, imposto por `UNIQUE (player_id, currency)` no schema.
- Todos os identificadores (`id` de wallet, transação, ledger entry, evento de outbox) são gerados pela **aplicação** (UUID v7, via `internal/infra/idgen`), nunca por `DEFAULT` do banco — necessário porque o mesmo id gerado em memória é reutilizado para montar entidades relacionadas (ex: o `transactionId` de uma `WagerTransaction` é referenciado pelo `WalletLedgerEntry` e pelos eventos de outbox, todos montados antes do primeiro `INSERT`). O schema mantém `DEFAULT gen_random_uuid()` como rede de segurança, embora nunca acionado no fluxo normal da aplicação.
- Cenários principais e testes usam exclusivamente BRL, conforme permitido pelo desafio, mas o tipo `Money` não impõe essa restrição.
