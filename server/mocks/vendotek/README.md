# Mock Vendotek (temporary)

Temporary HTTP mock for payment terminal behavior during backend development.

## Run

From `server`:

```bash
go run ./mocks/vendotek
```

Default port is `8082`.
Mock tries to load variables from `.env` in current directory first, then from `mocks/vendotek/.env`.

## Environment variables

- `PORT` - HTTP port, default `8082`
- `GIN_MODE` - Gin mode (`debug` / `release`)
- `VENDOTEK_DEFAULT_SCENARIO` - `approved` | `declined` | `timeout` | `random`, default `approved`
- `VENDOTEK_AUTO_DELAY_MS` - delay before auto-finalization after start, default `1200`
- `VENDOTEK_RANDOM_DECLINE_PCT` - decline percent for `random`, default `20`

Example:

```bash
PORT=8082 VENDOTEK_DEFAULT_SCENARIO=random VENDOTEK_RANDOM_DECLINE_PCT=30 go run ./mocks/vendotek
```

## API

- `GET /healthz`
- `POST /sessions`
- `GET /sessions/:id`
- `POST /sessions/:id/start`
- `POST /sessions/:id/approve`
- `POST /sessions/:id/decline`
- `POST /sessions/:id/cancel`

## Quick test flow

### 1) Create session

```bash
curl -sS -X POST http://localhost:8082/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "externalTransactionId":"tx_123",
    "amountMinor": 25000,
    "currency": "RUB"
  }'
```

### 2) Start processing

```bash
curl -sS -X POST http://localhost:8082/sessions/<sessionId>/start
```

### 3) Poll status

```bash
curl -sS http://localhost:8082/sessions/<sessionId>
```

### 4) Manual override (priority over auto rules)

```bash
curl -sS -X POST http://localhost:8082/sessions/<sessionId>/decline
```

Other options:

```bash
curl -sS -X POST http://localhost:8082/sessions/<sessionId>/approve
curl -sS -X POST http://localhost:8082/sessions/<sessionId>/cancel
```
