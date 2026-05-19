# Mock Vendotek EzPOS

EzPOS-совместимый мок платёжного терминала Vendotek для разработки и тестирования.
Реализует подмножество протокола EzPOS v1.5 (HTTP/JSON), которое использует адаптер сервера.
Является drop-in заменой реального терминала: чтобы переключиться на реальное устройство,
достаточно сменить `VENDOTEK_BASE_URL` в `server/.env` — код не меняется.

## Запуск

Из директории `server/`:

```bash
go run ./mocks/vendotek
```

Порт по умолчанию — `8082`. Сначала читается `server/.env`, затем `mocks/vendotek/.env`.

## Переменные окружения

| Переменная | Умолчание | Описание |
|---|---|---|
| `PORT` | `8082` | HTTP-порт мока |
| `VENDOTEK_DEFAULT_SCENARIO` | `success` | Сценарий операции: `success`, `decline`, `timeout`, `reverted` |
| `VENDOTEK_AUTO_WAIT_MS` | `500` | Задержка перехода в `wait_for_card` после создания операции |
| `VENDOTEK_AUTO_DELAY_MS` | `1500` | Задержка перехода в финальное состояние после `in_progress` |
| `VENDOTEK_TIMEOUT_MS` | `600000` | Задержка при сценарии `timeout` (10 мин) |
| `VENDOTEK_RANDOM_DECLINE_PCT` | `0` | Процент автоматических отказов (при значении > 0 переопределяет сценарий) |
| `VENDOTEK_SERIAL_NUMBER` | `MOCKVTK0001` | Серийный номер, возвращаемый в `GET /status` |
| `MOCK_VENDOTEK_DEBUG` | `false` | Verbose-логирование запросов/ответов |

## EzPOS эндпоинты

### `POST /async/cashless/sale`
### `POST /async/cashless/sale/card`
### `POST /async/cashless/sale/qr`

Создаёт новую операцию оплаты. Тело:

```json
{ "id": "abc123def456...", "amount": 25000, "currency": "RUB" }
```

Ответ `201 Created`:

```json
{ "id": "abc123def456...", "status": "created" }
```

После `VENDOTEK_AUTO_WAIT_MS` мс операция переходит в `wait_for_card`, затем — в `in_progress`,
затем через `VENDOTEK_AUTO_DELAY_MS` мс — в финальное состояние по сценарию.

---

### `GET /sale?id=<opId>`

Возвращает текущий статус операции.

Ответ `200 OK`:

```json
{
  "id": "abc123def456...",
  "status": "completed",
  "slip": {
    "pan": "411111******1111",
    "rrn": "123456789012",
    "approval_code": "ABC123",
    "amount": 25000,
    "date": "260518",
    "pos_entry_mode": "07",
    "app_label": "VISA"
  }
}
```

Поле `slip` присутствует только при `completed` и `reverted`.

Возможные статусы: `created`, `wait_for_card`, `in_progress`, `completed`, `reverted`, `fail`.

---

### `POST /async/cashless/sale/cancel?id=<opId>`

Отменяет операцию. Если операция в состоянии `wait_for_card` — переводит в `fail`.
Согласно спецификации EzPOS успех запроса не гарантирует прерывание — адаптер учитывает это.

Ответ `200 OK`: `{}` (пустой JSON).

---

### `POST /async/cashless/reversal?id=<opId>`

Инициирует возврат. Если операция `completed` — переводит в `reverted` с задержкой.

---

### `POST /async/fiscal?id=<opId>`

Возвращает `405 Method Not Allowed` — фискализация на стороне терминала не используется
(вариант B: фискализация выполняется нашей KKT-интеграцией).

---

### `GET /status`

Статус терминала для панели администратора.

Ответ `200 OK`:

```json
{
  "status": "ok",
  "S/N": "MOCKVTK0001",
  "info": "mock vendotek ready",
  "last_op_id": "abc123def456..."
}
```

---

### `POST /show/qr`, `POST /screen`

Заглушки, возвращают `200 OK` (не используются в текущем сценарии).

---

## Debug-эндпоинты

Для ручного управления в ходе разработки. **Не являются частью EzPOS** — адаптер их не вызывает.

```bash
# Подтвердить операцию вручную
curl -X POST http://localhost:8082/debug/ops/<id>/approve

# Отклонить
curl -X POST http://localhost:8082/debug/ops/<id>/decline

# Перевести в reverted
curl -X POST http://localhost:8082/debug/ops/<id>/reverted

# Отменить (cancel)
curl -X POST http://localhost:8082/debug/ops/<id>/cancel
```

## Сценарии

| `VENDOTEK_DEFAULT_SCENARIO` | Финальный статус | Когда использовать |
|---|---|---|
| `success` | `completed` | Золотой путь оплаты |
| `decline` | `fail` | Тест отклонения карты |
| `reverted` | `reverted` | Тест отмены/возврата |
| `timeout` | `fail` | Тест таймаута ожидания карты |

## Быстрый тест

```bash
# 1. Создать операцию (id — 32 hex-символа, как формирует адаптер)
curl -sS -X POST http://localhost:8082/async/cashless/sale \
  -H "Content-Type: application/json" \
  -d '{"id":"aabbccddeeff00112233445566778899","amount":25000,"currency":"RUB"}'

# 2. Опросить статус
curl -sS "http://localhost:8082/sale?id=aabbccddeeff00112233445566778899"

# 3. Подождать ~2 сек и опросить снова — ожидается completed + slip
curl -sS "http://localhost:8082/sale?id=aabbccddeeff00112233445566778899"
```
