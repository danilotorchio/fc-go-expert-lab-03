# Leilões com fechamento automático — Go Expert

Sistema de leilões (auctions) e lances (bids) em Go com MongoDB, baseado em [devfullcycle/labs-auction-goexpert](https://github.com/devfullcycle/labs-auction-goexpert), com **fechamento automático de leilões via goroutine**.

## Como funciona o fechamento automático

Ao criar um leilão, `AuctionRepository.CreateAuction` (`internal/infra/database/auction/create_auction.go`) grava o leilão no MongoDB e dispara uma goroutine que:

1. aguarda até `timestamp de criação + AUCTION_INTERVAL`, sem bloquear a requisição;
2. atualiza o status do leilão de `Active` (`0`) para `Completed` (`1`) no banco.

A mesma variável `AUCTION_INTERVAL` já é usada pela rotina de criação de lances para recusar lances em leilões expirados, então o fechamento e a validação dos lances usam o mesmo horário de término.

## Plus: reagendamento dos leilões abertos ao iniciar

Além do que o desafio pede: como o agendamento do fechamento vive em goroutines (em memória), um restart da aplicação faria os leilões abertos nunca serem fechados. Para evitar isso, ao iniciar a aplicação `AuctionRepository.ScheduleOpenAuctionsClosing` busca todos os leilões com status `Active` no MongoDB e dispara uma goroutine de fechamento para cada um:

- leilões que **já expiraram** enquanto a aplicação estava fora do ar são fechados imediatamente;
- leilões **ainda em andamento** são fechados no horário original (`timestamp de criação + AUCTION_INTERVAL`).

Para ver funcionando, crie um leilão, reinicie a aplicação antes do fim do `AUCTION_INTERVAL` (`docker compose restart app`) e consulte o leilão após o tempo configurado: o status muda para `1` normalmente.

## Variáveis de ambiente

Ficam em `cmd/auction/.env`:

| Variável | Descrição | Padrão no `.env` |
|---|---|---|
| `AUCTION_INTERVAL` | Duração do leilão até o fechamento automático (formato [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration): `30s`, `5m`, `1h`...). Se ausente ou inválida, usa `5m`. | `20s` |
| `BATCH_INSERT_INTERVAL` | Intervalo máximo para gravar o lote de lances | `20s` |
| `MAX_BATCH_SIZE` | Quantidade de lances por lote | `4` |
| `MONGODB_URL` | URL de conexão com o MongoDB | `mongodb://admin:admin@mongodb:27017/auctions?authSource=admin` |
| `MONGODB_DB` | Nome do banco | `auctions` |
| `MONGO_INITDB_ROOT_USERNAME` / `MONGO_INITDB_ROOT_PASSWORD` | Credenciais do container do MongoDB | `admin` / `admin` |

Para mudar a duração dos leilões, altere `AUCTION_INTERVAL` no `.env` e suba novamente os containers.

## Rodando com Docker Compose

```bash
docker compose up --build
```

A API fica disponível em `http://localhost:8080` e o MongoDB em `localhost:27017`.

## Testando o fechamento manualmente

Crie um leilão (`condition`: `0` novo, `1` usado, `2` recondicionado):

```bash
curl -X POST localhost:8080/auction \
  -H 'Content-Type: application/json' \
  -d '{"product_name":"Celular","category":"Eletronicos","description":"Celular novo na caixa lacrado","condition":1}'
```

Liste os leilões abertos (`status=0`) para obter o `id`:

```bash
curl 'localhost:8080/auction?status=0'
```

Consulte o leilão logo após a criação (`"status":0`) e novamente depois de `AUCTION_INTERVAL` (`"status":1`):

```bash
curl localhost:8080/auction/<id>
```

## Endpoints

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/auction` | Cria um leilão |
| `GET` | `/auction?status=&category=&productName=` | Lista leilões |
| `GET` | `/auction/:auctionId` | Busca um leilão |
| `GET` | `/auction/winner/:auctionId` | Lance vencedor do leilão |
| `POST` | `/bid` | Cria um lance (`user_id`, `auction_id`, `amount`) |
| `GET` | `/bid/:auctionId` | Lances de um leilão |
| `GET` | `/user/:userId` | Busca um usuário |

## Testes

Os testes em `internal/infra/database/auction/create_auction_test.go` usam um MongoDB real, com `AUCTION_INTERVAL=1s`:

- `TestCreateAuctionClosesAutomatically` — cenário do desafio:
  1. cria um leilão e verifica que o status é `Active`;
  2. aguarda o tempo configurado em `AUCTION_INTERVAL`;
  3. verifica que o status mudou para `Completed` sem nenhuma intervenção manual.
- `TestScheduleOpenAuctionsClosing` — plus: simula leilões abertos deixados por uma execução anterior (um já expirado e um em andamento), executa o reagendamento e verifica que o expirado é fechado imediatamente e o em andamento só após `AUCTION_INTERVAL`.

Cada teste usa um banco temporário, removido ao final. Sem `MONGODB_URL` definida, os testes são ignorados.

Via Docker (com o MongoDB do compose):

```bash
docker compose up -d mongodb
docker compose run --rm --entrypoint go app test ./... -v
```

Ou direto com Go (requer Go 1.20+):

```bash
docker compose up -d mongodb
MONGODB_URL='mongodb://admin:admin@localhost:27017/?authSource=admin' go test ./... -v
```
