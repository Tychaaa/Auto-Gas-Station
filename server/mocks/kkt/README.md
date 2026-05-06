# Mock онлайн-кассы PayOnline-01-ФА

TCP-эмулятор онлайн-кассы для разработки фискализации без доступа к реальному
устройству. Говорит по протоколу ККТ A.2.0 "стандартный нижний уровень"
(ENQ/ACK/NAK + STX/LEN/CMD/DATA/LRC), как и реальная PayOnline-01-ФА.

Документация на реальный протокол:
[paykiosk.ru/payonline-01-fa/payonline-01-fa-dokumentatsiya](https://www.paykiosk.ru/payonline-01-fa/payonline-01-fa-dokumentatsiya).

## Что эмулирует

Только тот минимум команд, который вызывает текущий адаптер
`server/internal/adapter/fiscal`:

- `0x10` — Короткий запрос состояния (ShortStatus). Возвращает успех и в флагах
  отдаёт "смена открыта" (бит 13), если сценарий не переопределил состояние.
- `0xFF40` — Запрос параметров текущей смены. Состояние смены настраивается через
  `MOCK_KKT_SHIFT_STATE` или сценарий `shift_*`.
- `0xFF46` — Операция V2 (приход). По умолчанию успех; сценарий
  `operation_error` возвращает код ошибки `0x49`.
- `0xFF45` — Закрытие чека V2. По умолчанию возвращает успех, сдачу `0`,
  инкрементальные `FDNumber` / `FiscalSign` и текущие дату-время. Сценарий
  `close_error` возвращает код ошибки `0xA0`.

Любую другую команду мок отклоняет кодом `0x42` ("команда не поддерживается").

## Запуск

Из директории `server`:

```bash
go run ./mocks/kkt
```

По умолчанию слушает `127.0.0.1:7778`.
Мок ищет `.env` в `server/mocks/kkt/.env`, затем `mocks/kkt/.env`, затем `.env`.

## Переменные окружения

Шаблон лежит рядом — `server/mocks/kkt/.env.example`.

| Переменная | По умолчанию | Описание |
| --- | --- | --- |
| `MOCK_KKT_HOST` | `127.0.0.1` | На каком интерфейсе слушать TCP. |
| `MOCK_KKT_PORT` | `7778` | TCP-порт. |
| `MOCK_KKT_SCENARIO` | `success` | `success` / `shift_closed` / `shift_expired` / `operation_error` / `close_error`. |
| `MOCK_KKT_SHIFT_STATE` | `open` | `open` / `closed` / `expired`. Перекрывается сценарием `shift_*`. |
| `MOCK_KKT_SHIFT_NUMBER` | `1` | Стартовый номер смены в ответе FF40. |
| `MOCK_KKT_RECEIPT_NUMBER` | `1` | Стартовый номер чека в ответе FF40. |
| `MOCK_KKT_DUMP_HEX` | `false` | Логировать hex-дампы кадров и служебных байт. |

## Подключение основного сервера

В `server/.env` указать адрес мока вместо реальной кассы:

```
KKT_HOST=127.0.0.1
KKT_PORT=7778
```

Прочие `KKT_*` параметры (пароли, СНО, ставка НДС, признаки расчёта) можно
оставить как в `server/.env.example` — мок их не валидирует, но текущий
адаптер всё равно проверит формат локально.

## Сценарии

| Сценарий | Что проверяем в основном сервере |
| --- | --- |
| `success` | Полный happy path: транзакция уходит в `paid` + `FiscalStatus=done`, в результате `FDNumber` / `FiscalSign` / `ReceiptNumber` заполнены. |
| `shift_closed` | `Adapter.Fiscalize` падает с `ErrKindShiftClosed`, транзакция помечается `failed`. |
| `shift_expired` | То же, но смена просрочена (>24ч). |
| `operation_error` | `OperationV2` возвращает ошибку — `ErrKindOperationFailed`. |
| `close_error` | `CloseReceiptV2` возвращает ошибку — `ErrKindCloseFailed`. |

Сменить сценарий можно перезапуском мока с другим значением `MOCK_KKT_SCENARIO`.
