# Примеры els-go

Запускаемые примеры для [ELS Go SDK](../README_RU.md). Каждая папка —
отдельный модуль (свой `go.mod`), который собирается против локального SDK.

> 🇬🇧 [English version → README.md](README.md)

## Запуск

```bash
cd ru/minimal-quickstart
ELS_API_KEY=els_live_xxxxxxxx go run .
```

Примеры с английскими комментариями — в `en/`, с русскими — в `ru/`,
код идентичен.

## Сценарии

| Пример | Что показывает |
|---|---|
| `minimal-quickstart` | Минимальная настройка — только `APIKey` + `AppSlug` |
| `basic` | Ошибки, сообщения, синхронная отправка, graceful shutdown |
| `capture-error-vs-message` | `CaptureError` vs `CaptureMessage` vs `SendSync` |
| `level-shortcuts` | Хелперы `Debug/Info/Warning/Error/Critical` |
| `levels` | Типизированный `Level` + маппинг slog↔ELS |
| `slog` | Интеграция `log/slog`; ошибки уходят со стек-трейсом |
| `context` | Проброс request/trace ID через `context.Context` |
| `service-defaults` | `ServiceName`/`AppSlug`/env один раз на клиенте |
| `filtering-sampling` | `MinLevel` + `SampleRate` (critical всегда проходит) |
| `before-send-redaction` | Хук `BeforeSend` для маскировки PII / отбрасывания |
| `worker-shutdown` | Батчинг + graceful shutdown по SIGINT/SIGTERM |
| `disk-buffer` | Офлайн-устойчивость через дисковый буфер |
| `user-context` | Привязка данных пользователя к каждой записи |
| `health-check` | `Health()` как readiness-проба |
| `global-facade` | Package-level `Init` + хелперы `*Global` |
| `custom-http-client` | Передача своего `*http.Client` |
| `middleware` | net/http middleware для перехвата паник |
