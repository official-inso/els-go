# ELS Go SDK

Высокопроизводительный асинхронный Go SDK для [Error Logs Service (ELS)](https://github.com/official-inso/els-go).

[English version](README.md)

## Возможности

- **Без внешних зависимостей** — только стандартная библиотека Go
- **Асинхронная пакетная отправка** — фоновая горутина собирает ошибки и отправляет пачками
- **Синхронная отправка** — `SendSync()` для критичных ошибок с подтверждением доставки
- **Автоматические повторы** — экспоненциальная задержка с обработкой 429 rate-limit
- **Типизированные ошибки** — различение retryable/permanent через `IsRetryableErr()`
- **Буферизация на диск** — неотправленные записи сохраняются и повторно отправляются при следующем запуске
- **Фильтрация по уровню** — отбрасывание debug/info в production через `MinLevel`
- **Middleware для panic recovery** — автоматический перехват паник в HTTP-хендлерах
- **Health check** — проверка доступности сервера ELS
- **Хук BeforeSend** — фильтрация или модификация записей перед отправкой
- **Трекинг сессий** — автоматический session ID для корреляции связанных ошибок
- **Корректное завершение** — `Close()` опустошает очередь и сохраняет остаток

## Установка

```bash
go get github.com/official-inso/els-go
```

## Быстрый старт

```go
package main

import (
    "context"
    "errors"
    "log"

    els "github.com/official-inso/els-go"
)

func main() {
    client, err := els.New(els.Config{
        Endpoint:      "https://api.example.com/els",
        APIKey:        "ваш-api-ключ",
        AppSlug:       "my-service",
        DeploymentEnv: "PRODUCTION",
        ServiceName:   "api-gateway",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Асинхронный захват (неблокирующий)
    client.CaptureError(errors.New("что-то пошло не так"),
        els.WithURL("/api/users"),
        els.WithLevel(els.LevelError),
    )

    // Синхронная отправка для критичных ошибок (блокирует до подтверждения)
    ctx := context.Background()
    if err := client.SendSync(ctx, errors.New("ошибка платежа"),
        els.WithURL("/api/pay"),
        els.WithLevel(els.LevelCritical),
    ); err != nil {
        log.Printf("не удалось отправить: %v", err)
    }
}
```

## Конфигурация

| Поле | Тип | По умолчанию | Описание |
|------|-----|--------------|----------|
| `Endpoint` | string | *обязательно* | Базовый URL ELS API |
| `APIKey` | string | *обязательно* | API-ключ для аутентификации |
| `AppSlug` | string | `""` | Идентификатор приложения |
| `DeploymentEnv` | string | `""` | Окружение (DEV, STAGING, PRODUCTION) |
| `ServiceName` | string | `""` | Название микросервиса |
| `BatchSize` | int | `50` | Макс. записей в одном запросе |
| `BatchInterval` | Duration | `5s` | Макс. время до отправки неполной пачки |
| `BufferSize` | int | `1000` | Ёмкость очереди в памяти |
| `MaxRetries` | int | `3` | Количество повторных попыток |
| `RetryBaseDelay` | Duration | `1s` | Начальная задержка (удваивается) |
| `Timeout` | Duration | `10s` | Таймаут HTTP-запроса |
| `FlushTimeout` | Duration | `10s` | Макс. время ожидания Flush() |
| `BufferDir` | string | `os.TempDir()` | Директория для файла буфера |
| `MaxBufferFileSize` | int64 | `100MB` | Макс. размер файла буфера |
| `MinLevel` | string | `""` | Минимальный уровень для захвата |
| `BeforeSend` | func | `nil` | Хук для фильтрации/мутации |
| `OnError` | func | `nil` | Коллбек внутренних ошибок |
| `DefaultLevel` | string | `"error"` | Уровень по умолчанию |
| `DefaultSource` | string | `"server"` | Источник по умолчанию |
| `Debug` | bool | `false` | Отладочный вывод |

## Синхронная отправка

Для критичных ошибок, где доставка должна быть подтверждена:

```go
err := client.SendSync(ctx, errors.New("ошибка платежа"),
    els.WithURL("/api/payments"),
    els.WithLevel(els.LevelCritical),
)
if err != nil {
    if els.IsRetryableErr(err) {
        // Проблема сервера/сети — можно повторить позже
    } else {
        // Постоянная ошибка (авторизация, валидация) — не повторять
    }
}
```

## Фильтрация по уровню

Отбрасывание малозначимых записей в production:

```go
client, _ := els.New(els.Config{
    // ...
    MinLevel: els.LevelWarning, // отбрасывает debug и info
})
```

Приоритет уровней: `debug` < `info` < `warning` < `error` < `critical`

## Health Check

```go
if err := client.Health(ctx); err != nil {
    log.Printf("ELS недоступен: %v", err)
}
```

## HTTP Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/health", healthHandler)

// Вариант 1: Перехватывает панику и пробрасывает дальше
handler := client.Middleware(mux)

// Вариант 2: Перехватывает панику и возвращает 500 (standalone)
handler := client.RecoverMiddleware(mux)

http.ListenAndServe(":8080", handler)
```

## Опции захвата

```go
client.CaptureError(err,
    els.WithLevel(els.LevelCritical),
    els.WithURL("/api/orders/123"),
    els.WithSource(els.SourceServer),
    els.WithUserAgent("CustomBot/1.0"),
    els.WithLanguage("ru-RU"),
    els.WithReferrer("http://example.com"),
    els.WithMeta(map[string]any{"orderId": "123"}),
    els.WithSessionID("custom-session-id"),
    els.WithServiceName("payment-worker"),
    els.WithStack(customStackTrace),
    els.WithComponentStack(reactTrace),
)
```

## Буферизация на диск

Когда ELS-сервер недоступен, записи сохраняются в `.els-buffer.jsonl`. При следующем запуске буфер автоматически отправляется. Файл ограничен `MaxBufferFileSize` (по умолчанию 100MB) — при превышении новые записи отбрасываются.

## Корректное завершение

```go
client, _ := els.New(config)
defer client.Close()
```

`Close()`:
1. Прекращает приём новых записей
2. Отправляет все оставшиеся записи в очереди
3. Сохраняет неотправленные записи на диск

## Типизированные ошибки

Все операции отправки возвращают `*SendError`:

```go
err := client.SendSync(ctx, myErr)
var sendErr *els.SendError
if els.As(err, &sendErr) {
    fmt.Printf("Статус: %d, Повторяемая: %v\n", sendErr.StatusCode, sendErr.IsRetryable)
}
```

## Описание полей

См. [docs/FIELDS_RU.md](docs/FIELDS_RU.md) для подробного описания всех полей.

## Примеры

- [Базовый пример (RU)](examples/ru/basic/main.go)
- [HTTP middleware (RU)](examples/ru/middleware/main.go)
- [Basic usage (EN)](examples/en/basic/main.go)
- [HTTP middleware (EN)](examples/en/middleware/main.go)

## Лицензия

MIT
