# Leilões com fechamento automático — Go Expert

Sistema de leilões (auctions) e lances (bids) em Go com MongoDB, baseado em [devfullcycle/labs-auction-goexpert](https://github.com/devfullcycle/labs-auction-goexpert), com **fechamento automático de leilões via goroutine**.

## Como funciona o fechamento automático

Ao criar um leilão, `AuctionRepository.CreateAuction` (`internal/infra/database/auction/create_auction.go`) grava o leilão no MongoDB e dispara uma goroutine que:

1. aguarda até `timestamp de criação + AUCTION_INTERVAL`, sem bloquear a requisição;
2. atualiza o status do leilão de `Active` (`0`) para `Completed` (`1`) no banco.

A mesma variável `AUCTION_INTERVAL` já é usada pela rotina de criação de lances para recusar lances em leilões expirados, então o fechamento e a validação dos lances usam o mesmo horário de término.

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

O teste `TestCreateAuctionClosesAutomatically` (`internal/infra/database/auction/create_auction_test.go`) usa um MongoDB real e valida o cenário:

1. cria um leilão e verifica que o status é `Active`;
2. aguarda o tempo configurado em `AUCTION_INTERVAL` (`1s` no teste);
3. verifica que o status mudou para `Completed` sem nenhuma intervenção manual.

Cada execução usa um banco temporário, removido ao final. Sem `MONGODB_URL` definida, o teste é ignorado.

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
