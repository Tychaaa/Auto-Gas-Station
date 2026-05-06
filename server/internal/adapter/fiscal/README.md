# Фискализация (PayOnline-01-ФА)

Этот пакет инкапсулирует работу с ККТ PayOnline-01-ФА по протоколу ККТ A.2.0
(TCP, стандартный нижний уровень STX/LRC + ENQ/ACK/NAK). Низкоуровневая часть —
в подпакете [`kkt`](./kkt).

## Когда срабатывает

Чек формируется автоматически **сразу после успешной оплаты, до старта налива**:

```
selection -> payment_pending -> paid -> fiscalizing -> paid + fiscal=done -> fueling -> completed
```

Триггер — `service.PaymentService` после получения статуса `approved` от
платёжного адаптера (`Vendotek`). Вызов идёт в `service.FiscalService.FiscalizePaid`,
который:

1. забирает свежий snapshot транзакции (`paid` + `FiscalStatus=none|failed`);
2. собирает доменный `ReceiptInput` из транзакции;
3. переводит транзакцию в `fiscalizing` под mutex хранилища;
4. вызывает `Adapter.Fiscalize` (TCP к ККТ) — **без удержания mutex**;
5. при успехе — `MarkPaidFiscalized(receipt)` (Status снова `paid`,
   `FiscalStatus=done`, заполнен `ReceiptNumber`);
6. при ошибке — `MarkFiscalFailed(msg)` (Status `failed`, `FiscalError`).

Запустить отпуск топлива можно только из `paid` с `FiscalStatus=done`
(см. `Transaction.BeginFueling`).

## Источник данных чека

Поле `ReceiptInput` собирается из `model.Transaction`:

| Поле               | Источник                                                                |
| ------------------ | ----------------------------------------------------------------------- |
| `GoodName`         | `tx.FuelType`                                                           |
| `UnitPriceMinor`   | `tx.UnitPriceMinor`                                                     |
| `TotalMinor`       | `tx.ComputedAmountMinor`                                                |
| `QuantityMicro`    | `tx.Liters * 1e6`, иначе `round(ComputedAmountMinor * 1e6 / UnitPrice)` |
| `PaymentKind`      | `cashless` для текущего терминала Vendotek                              |
| `RoundingMinor`    | `0` (округление до рубля не применяем)                                  |

## Конфигурация (env)

См. `server/.env.example`. Типовые значения по умолчанию под УСН «доходы минус расходы»
и полный расчёт по предоплаченному заказу:

- `KKT_TAX_SYSTEM=USN_INCOME_EXPENSE` — УСН доход-расход (бит 2 в FF45h);
- `KKT_VAT_RATE=NO_VAT` — без НДС (0x08 в FF46h);
- `KKT_PAYMENT_METHOD_SIGN=4` — полный расчёт (тег 1214);
- `KKT_PAYMENT_SUBJECT_SIGN=1` — товар (тег 1212);
- в коде поддерживаются СНО без устаревшего ЕНВД; код `ESHN` соответствует ЕСХН.

Пароли по умолчанию `30 / 30` совпадают с дефолтами PayOnline и берутся из env
`KKT_SYSADMIN_PASSWORD` / `KKT_OPERATOR_PASSWORD`.

## Ограничения

- Смену открывает/закрывает оператор вручную через FN-мастер или драйвер ККТ.
  Если смена закрыта или просрочена (>24ч), фискализация падает с ошибкой
  `ErrKindShiftClosed`, транзакция уходит в `failed`, оплата не повторяется.
- Идентификатор транзакции в чек не передаётся — в ККТ уходит только позиция по топливу.
- Адаптер не потокобезопасен: на каждом вызове `Fiscalize` открывается отдельное
  TCP-соединение и закрывается после обмена.

## Тесты

- `kkt/*_test.go` — кодирование/декодирование кадров и низкоуровневых полей.
- `service/fiscal_service_test.go` — построение `ReceiptInput` для всех
  `OrderMode`, успех/ошибка адаптера, проверка отказа от не-paid транзакции.
- `model/transaction_fiscal_test.go` — переходы `paid -> fiscalizing -> paid`
  и `paid -> fueling -> completed`.
