# ELS Go SDK

Высокопроизводительный Go SDK для [Error Logs Service](https://github.com/official-inso/els-go). Без зависимостей, асинхронная пакетная отправка, буферизация на диск.

[English version](README.md)

## Установка

```bash
go get github.com/official-inso/els-go
```

## Быстрый старт

```go
package main

import (
    "errors"
    "log"

    els "github.com/official-inso/els-go"
)

func main() {
    // Вариант A: глобальный клиент (рекомендуется для большинства приложений)
    els.Init(els.Config{
        Endpoint:      "https://api.example.com/els",
        APIKey:        "ваш-api-ключ",
        AppSlug:       "my-service",
        DeploymentEnv: "PRODUCTION",
    })
    defer els.Close()

    els.CaptureErrorGlobal(errors.New("что-то сломалось"), els.WithURL("/api/users"))

    // Вариант B: явный клиент (для библиотек или нескольких экземпляров)
    client, err := els.New(els.Config{...})
    if err != nil { log.Fatal(err) }
    defer client.Close()

    client.CaptureError(errors.New("таймаут БД"), els.WithURL("/api/data"))
}
```

## Основные концепции

### Async vs Sync

Большинство вызовов **асинхронны** — возвращаются мгновенно, отправка в фоне:

```go
client.CaptureError(err, els.WithURL("/api"))       // неблокирующий
client.CaptureMessage("запущен", els.LevelInfo)     // неблокирующий
```

Для критичных ошибок с **подтверждением доставки** — `SendSync`:

```go
err := client.SendSync(ctx, paymentErr,
    els.WithURL("/api/pay"),
    els.WithLevel(els.LevelCritical),
)
if err != nil {
    // Ошибка НЕ доставлена
}
```

### Паттерн опций

Каждый метод захвата принимает опции для обогащения записи:

```go
client.CaptureError(err,
    els.WithURL("/api/orders"),
    els.WithLevel(els.LevelCritical),
    els.WithMeta(map[string]any{"orderId": "123"}),
    els.WithRequest(httpReq),        // авто-извлекает URL, UA, Referrer, заголовки
    els.WithCause(err),              // сохраняет цепочку ошибок в meta
    els.WithSessionID("req-abc"),
)
```

Полный список: `WithURL`, `WithLevel`, `WithSource`, `WithStack`, `WithMeta`, `WithUserAgent`, `WithLanguage`, `WithReferrer`, `WithSessionID`, `WithServiceName`, `WithComponentStack`, `WithRequest`, `WithCause`.

---

## Возможности

### HTTP Middleware

Автоматический перехват паник в HTTP-хендлерах:

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/data", handler)

// Перехватывает панику + возвращает 500
http.ListenAndServe(":8080", client.RecoverMiddleware(mux))

// Перехватывает панику + пробрасывает дальше (если есть свой recovery)
http.ListenAndServe(":8080", client.Middleware(mux))
```

### Контекст пользователя

Привязка информации о пользователе ко всем последующим записям:

```go
client.SetUser(&els.UserContext{
    ID:    "usr_123",
    Email: "john@example.com",
    Extra: map[string]string{"tenant": "acme"},
})
// Все записи теперь содержат user.id, user.email, user.tenant в Meta
```

### Интеграция с slog

Направление стандартного `log/slog` в ELS:

```go
logger := slog.New(els.SlogHandler(client, nil))
slog.SetDefault(logger)

slog.Error("таймаут БД", "host", "pg-1", "latency_ms", 5200)
// → захвачено как ошибка ELS с meta: {"host": "pg-1", "latency_ms": 5200}
```

### Семплирование

Контроль объёма в высоконагруженном production:

```go
els.New(els.Config{
    SampleRate: 0.1, // отправляет ~10% некритичных записей
})
```

Critical-уровень всегда проходит независимо от SampleRate.

### Фильтрация по уровню

Полное отбрасывание малозначимых записей:

```go
els.New(els.Config{
    MinLevel: els.LevelWarning, // отбрасывает debug и info
})
```

Приоритет: `debug` < `info` < `warning` < `error` < `critical`

### Health Check

```go
if err := client.Health(ctx); err != nil {
    log.Printf("ELS недоступен: %v", err)
}
```

### Буферизация на диск

При недоступности сервера после ретраев записи сохраняются в `.els-buffer.jsonl`. При следующем запуске автоматически отправляются. Лимит — `MaxBufferFileSize` (по умолчанию 100MB).

### Метрики

```go
stats := client.GetStats()
log.Printf("queued=%d sent=%d dropped=%d failed=%d sampled=%d queue=%d",
    stats.Enqueued, stats.Sent, stats.Dropped, stats.Failed, stats.Sampled,
    client.QueueSize(),
)
```

### Типизированные ошибки

Различение retryable и permanent ошибок:

```go
err := client.SendSync(ctx, myErr)
if els.IsRetryableErr(err) {
    // 5xx, 429, сеть — можно повторить
} else {
    // 4xx — не повторять (авторизация, валидация)
}
```

---

## Конфигурация

```go
els.Config{
    // Обязательные
    Endpoint string    // Базовый URL ELS API
    APIKey   string    // API-ключ

    // Идентификация (рекомендуется)
    AppSlug       string // Идентификатор приложения
    DeploymentEnv string // DEV, STAGING, PRODUCTION
    ServiceName   string // Название микросервиса

    // Батчинг
    BatchSize     int           // Макс. записей в запросе (по умолчанию: 50)
    BatchInterval time.Duration // Интервал отправки (по умолчанию: 5s)
    BufferSize    int           // Ёмкость очереди (по умолчанию: 1000)

    // Ретраи
    MaxRetries     int           // Попытки (по умолчанию: 3)
    RetryBaseDelay time.Duration // Начальная задержка (по умолчанию: 1s)
    Timeout        time.Duration // HTTP таймаут (по умолчанию: 10s)

    // Буферизация
    BufferDir         string // Директория буфера (по умолчанию: os.TempDir())
    MaxBufferFileSize int64  // Макс. размер файла (по умолчанию: 100MB)

    // Фильтрация
    MinLevel   string  // Минимальный уровень (по умолчанию: все)
    SampleRate float64 // 0.0-1.0, critical всегда проходит (по умолчанию: 1.0)

    // Хуки
    BeforeSend func(*ErrorEntry) *ErrorEntry // Фильтрация/мутация
    OnError    func(error)                   // Коллбек ошибок SDK

    // Расширенные
    HTTPClient    *http.Client  // Свой HTTP-клиент
    FlushTimeout  time.Duration // Таймаут Flush() (по умолчанию: 10s)
    DefaultLevel  string        // По умолчанию: "error"
    DefaultSource string        // По умолчанию: "server"
    Debug         bool          // Отладочный вывод
}
```

---

## Корректное завершение

```go
defer client.Close() // или defer els.Close() для глобального
```

`Close()`: прекращает приём → опустошает очередь → отправляет → буферизует на диск.

---

## Примеры

- [Базовый пример (RU)](examples/ru/basic/main.go) — инициализация, захват, sync send, health
- [HTTP middleware (RU)](examples/ru/middleware/main.go) — перехват паник, ручной захват
- [Basic usage (EN)](examples/en/basic/main.go)
- [HTTP middleware (EN)](examples/en/middleware/main.go)

## Описание полей

См. [docs/FIELDS_RU.md](docs/FIELDS_RU.md) для подробного описания всех полей записи.

## Лицензия

MIT
