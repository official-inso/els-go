# ELS Go SDK

Высокопроизводительный Go SDK для **Error Logs Service (ELS)** — управляемого SaaS централизованного сбора событий (от debug до fatal) с AI-диагностикой ошибок. Без зависимостей, асинхронная пакетная отправка, буферизация на диск.

> 🇬🇧 [English version → README.md](README.md)

## Что вы получаете

ELS из коробки даёт встроенную админ-панель. Каждое событие, отправленное этим SDK, попадает туда — с полнотекстовым поиском, фасетной фильтрацией, AI-диагностикой и обнаружением регрессий по версиям.

| | |
|---|---|
| ![Список логов](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/01-error-logs-list.png) | ![Карточка события](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/02-event-detail-info.png) |
| Виртуальная таблица с фасетным сайдбаром (приложение, окружение, **версия**, источник, уровень, браузер, IP, категория). Live-режим обновляет данные каждые 5с. | Полные метаданные события: время, гео, окружение, **версия приложения**, fingerprint, session, карточки повторений, корреляция в рамках сессии. |
| ![AI-диагностика](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/03-error-detail-ai.png) | ![Аналитика](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/04-analytics-dashboard.png) |
| Распарсенный stack trace + AI-анализ: что сломалось, где, как чинить. | Timeline, donut'ы, топ URL/IP, тепловая карта по часам, **виджет регрессий по версиям**. |

Ещё нет API-ключа? **[Зарегистрируйтесь на lk.insoweb.ru](https://lk.insoweb.ru)** — займёт минуту.

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

## Миграция

### С go.uber.org/zap

**Было:**

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("user logged in", zap.Int("userId", 42))
logger.Error("payment failed", zap.Error(err))
```

**Стало:**

```go
import els "github.com/official-inso/els-go"

els.Init(els.Config{
    Endpoint:      "https://api.insoweb.ru/els",
    APIKey:        os.Getenv("ELS_API_KEY"),
    AppSlug:       "my-service",
    DeploymentEnv: "PRODUCTION",
    AppVersion:    os.Getenv("BUILD_VERSION"),
})
defer els.Close()

els.CaptureMessageGlobal("user logged in", els.LevelInfo,
    els.WithMeta(map[string]any{"userId": 42}))
els.CaptureErrorGlobal(err, els.WithURL("/api/pay"))
```

| zap | ELS | Заметки |
|---|---|---|
| `zap.NewProduction()` | `els.Init(els.Config{...})` | Один раз на старте |
| `logger.Info(msg, zap.X(...))` | `els.CaptureMessageGlobal(msg, level, els.WithMeta(...))` | Или используйте `els.SlogHandler` для `log/slog` |
| `zap.Error(err)` | `els.CaptureErrorGlobal(err, ...)` | Отдельный путь для ошибок |
| `logger.With(fields)` | `els.WithMeta(...)` per call | Или обёртка, прибивающая meta заранее |
| `logger.Sugar()` | не предоставляется | Оставайтесь структурированными |
| Sampling | `Config.SampleRate` | То же |

**Подводные камни:**

- Encoder-варианты zap (`console`, `json`) не настраиваются — wire-формат фиксированный JSON.
- Для drop-in совместимости — `slog` integration (см. Возможности); zap-вызовы ложатся естественно.

---

### С sirupsen/logrus

**Было:**

```go
import "github.com/sirupsen/logrus"

log := logrus.New()
log.SetFormatter(&logrus.JSONFormatter{})
log.SetLevel(logrus.InfoLevel)

log.WithFields(logrus.Fields{"userId": 42}).Info("user logged in")
log.WithError(err).Error("payment failed")
```

**Стало:**

```go
import els "github.com/official-inso/els-go"

els.Init(els.Config{
    Endpoint:   "https://api.insoweb.ru/els",
    APIKey:     os.Getenv("ELS_API_KEY"),
    AppSlug:    "my-service",
    MinLevel:   "info",
})
defer els.Close()

els.CaptureMessageGlobal("user logged in", els.LevelInfo,
    els.WithMeta(map[string]any{"userId": 42}))
els.CaptureErrorGlobal(err, els.WithURL("/api/pay"))
```

| logrus | ELS | Заметки |
|---|---|---|
| `logrus.New()` | `els.Init(els.Config{...})` | Глобальный или instance |
| `WithFields(Fields{...})` | `els.WithMeta(map[string]any{...})` | Та же форма |
| `WithError(err)` | `els.CaptureErrorGlobal(err)` | Отдельный путь |
| `SetLevel` | `Config.MinLevel` | То же |
| Hooks (`AddHook`) | `Config.BeforeSend` | Та же роль |
| Text formatter | не предоставляется | Только JSON на проводе |

**Подводные камни:**

- logrus в режиме maintenance. ELS совместим с современным `log/slog` — см. Возможности.
- `logrus.PanicLevel` ≈ ELS `LevelCritical`; SDK сам не паникует.

---

### С log/slog (стандартная библиотека)

`log/slog` интегрируется нативно — миграция не нужна, кроме направления handler в ELS.

**Было:**

```go
import "log/slog"
import "os"

logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
slog.SetDefault(logger)

slog.Info("user logged in", "userId", 42)
slog.Error("payment failed", "err", err)
```

**Стало:**

```go
import (
    "log/slog"
    els "github.com/official-inso/els-go"
)

client, _ := els.New(els.Config{
    Endpoint: "https://api.insoweb.ru/els",
    APIKey:   os.Getenv("ELS_API_KEY"),
    AppSlug:  "my-service",
})
defer client.Close()

logger := slog.New(els.SlogHandler(client, nil))
slog.SetDefault(logger)

slog.Info("user logged in", "userId", 42)
slog.Error("payment failed", "err", err)
```

| `log/slog` | ELS | Заметки |
|---|---|---|
| `slog.NewJSONHandler(os.Stderr, nil)` | `els.SlogHandler(client, nil)` | Тот же контракт handler |
| `slog.Info("msg", "k", v)` | без изменений | Едет в ELS как `info` |
| `slog.With(...)` | без изменений | Добавляет k/v в `meta` |
| Две цели (stdout + ELS) | Композитный handler | Оборачивайте оба через fan-out |

**Подводные камни:**

- Дефолтный уровень — `INFO`. Поставьте `MinLevel`, если нужен `DEBUG`.
- `slog.Error("msg", "err", err)` отправляет err как `meta.err`. Чтобы прийти с полным стеком — используйте `els.CaptureError(err, ...)` напрямую.

---

## Примеры

- [Базовый пример (RU)](examples/ru/basic/main.go) — инициализация, захват, sync send, health
- [HTTP middleware (RU)](examples/ru/middleware/main.go) — перехват паник, ручной захват
- [Basic usage (EN)](examples/en/basic/main.go)
- [HTTP middleware (EN)](examples/en/middleware/main.go)

## Описание полей

См. [docs/FIELDS_RU.md](docs/FIELDS_RU.md) для подробного описания всех полей записи.

## Почему ELS

ELS для Go — сфокусированный SaaS для логирования, а не observability-комбайн. Оптимизирован под скорость захвата, AI-диагностику и дешевизну интеграции.

- **Меньше веса.** Один модуль, без внешних зависимостей.
- **Ноль внешних API.** Только `POST /errors[/batch]` и `GET /health`.
- **AI-диагностика** на каждом stack trace, из коробки.
- **5 минут интеграции.** `go get` + `els.Init(...)` — готово.
- **Прозрачные тарифы.** Цены в личном кабинете.

### Подробное сравнение

| Категория | ELS | Sentry | Datadog / New Relic | Grafana Loki | LogRocket / Logtail / BetterStack |
|---|---|---|---|---|---|
| Модель хостинга | Managed SaaS | SaaS или self-hosted | Только SaaS | Self-hosted / Grafana Cloud | SaaS |
| Runtime-зависимости SDK | Ноль | Средне (саб-SDK, интеграции) | Тяжёлый агент + tracing | Promtail / агент | Средне |
| Время интеграции | ~5 мин | 10–20 мин | 30–60 мин | Часы — дни | 10–20 мин |
| AI-диагностика | Встроена | Платный аддон | Платный аддон | Нет | Нет |
| Группировка / fingerprint | Да | Да | Да | Вручную через LogQL | Частично |
| Source-map upload | Нет | Да | Да | н/п | Частично |
| Session replay (frontend) | Нет | Платно | Платно | н/п | Да (core) |
| Distributed tracing / APM | Нет | Частично | Да (core) | Да с Tempo | Нет |
| Метрики инфраструктуры | Нет | Нет | Да (core) | Да с Mimir | Нет |
| Хранение на free-тарифе | 24 часа | 30 дней (лимит объёма) | Только триал | Self-cost | 3–30 дней |
| Поддержка / документация на русском | Нативно | Сообщество | Ограничено | Сообщество | Нет |

### Когда ELS — неподходящий выбор

- Нужен один вендор на **APM + логи + метрики** одним счётом — берите Datadog или New Relic.
- Триаж фронтенда строится вокруг **DOM session replay** — LogRocket или Sentry Replay.
- Публичное мобильное приложение, нужны symbolication и ANR-детект — Firebase Crashlytics или Sentry Mobile.

Во всех остальных сценариях — backend-ошибки, JS-ошибки фронта, request-логи, структурированные события с version-aware-аналитикой — ELS даёт самый короткий путь до рабочей панели.

→ **Регистрация на [lk.insoweb.ru](https://lk.insoweb.ru)** для API-ключа.

## Другие ELS SDK

Тот же wire-формат, та же панель — выбирайте по стеку.

**Go** (этот репо)
- `github.com/official-inso/els-go` — основной SDK с `slog`-handler, HTTP middleware

**Node.js**
- [`@inso_web/els-client`](https://github.com/official-inso/els-client) — базовый TS / Node / browser клиент
- [`@inso_web/els-express`](https://github.com/official-inso/els-express) — Express middleware
- [`@inso_web/els-next`](https://github.com/official-inso/els-next) — хелперы Next.js
- [`@inso_web/els-nest`](https://github.com/official-inso/els-nest) — NestJS module
- [`@inso_web/els-react`](https://github.com/official-inso/els-react) — React Provider, hooks, ErrorBoundary
- [`@inso_web/els-vue`](https://github.com/official-inso/els-vue) — Vue 3 plugin

**Другие стеки**
- [`Inso.Els`](https://github.com/official-inso/els-csharp) — .NET (Core + ASP.NET Core + ILogger)
- [`io.github.official-inso:els-core`](https://github.com/official-inso/els-java) — Java + Spring Boot starter + SLF4J

## Тарифы

Free-тариф — **хранение логов 24 часа**. Полный прайс на **[lk.insoweb.ru](https://lk.insoweb.ru)**.

## Лицензия

MIT
