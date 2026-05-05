# ELS Go SDK

Высокопроизводительный асинхронный Go SDK для [Error Logs Service (ELS)](https://github.com/official-inso/els-go).

[English version](README.md)

## Возможности

- **Без внешних зависимостей** — только стандартная библиотека Go
- **Асинхронная пакетная отправка** — фоновая горутина собирает ошибки и отправляет пачками
- **Автоматические повторы** — экспоненциальная задержка с обработкой 429 rate-limit
- **Буферизация на диск** — неотправленные записи сохраняются и повторно отправляются при следующем запуске
- **Middleware для panic recovery** — автоматический перехват паник в HTTP-хендлерах
- **Хук BeforeSend** — фильтрация или модификация записей перед отправкой
- **Трекинг сессий** — автоматический session ID для корреляции связанных ошибок

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

    // Захват ошибки (асинхронно, неблокирующе)
    client.CaptureError(errors.New("таймаут подключения к базе данных"),
        els.WithURL("/api/users"),
        els.WithLevel(els.LevelCritical),
    )

    // Захват информационного сообщения
    client.CaptureMessage("сервис успешно запущен", els.LevelInfo,
        els.WithURL("/"),
    )
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
| `RetryBaseDelay` | Duration | `1s` | Начальная задержка между попытками (удваивается) |
| `Timeout` | Duration | `10s` | Таймаут HTTP-запроса |
| `BufferDir` | string | `os.TempDir()` | Директория для файла буфера |
| `BeforeSend` | func | `nil` | Хук для фильтрации/мутации записей |
| `OnError` | func | `nil` | Коллбек внутренних ошибок |
| `DefaultLevel` | string | `"error"` | Уровень серьёзности по умолчанию |
| `DefaultSource` | string | `"server"` | Источник по умолчанию |
| `Debug` | bool | `false` | Включить отладочный вывод |

## HTTP Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/health", healthHandler)

// Оборачиваем для автоматического перехвата паник
http.ListenAndServe(":8080", client.Middleware(mux))
```

## Опции захвата

Опции передаются в `CaptureError`, `CaptureMessage` и `CaptureEntry`:

```go
client.CaptureError(err,
    els.WithLevel(els.LevelCritical),
    els.WithURL("/api/orders/123"),
    els.WithSource(els.SourceServer),
    els.WithUserAgent("CustomBot/1.0"),
    els.WithMeta(map[string]any{"orderId": "123", "userId": "456"}),
    els.WithSessionID("custom-session-id"),
)
```

## Буферизация на диск

Когда ELS-сервер недоступен после всех повторов, записи сохраняются в файл `.els-buffer.jsonl` в директории `BufferDir` (по умолчанию — системная временная директория). При следующем запуске клиента буфер автоматически отправляется.

## Корректное завершение

Всегда вызывайте `client.Close()` перед завершением приложения:

```go
client, _ := els.New(config)
defer client.Close()
```

`Close()` выполнит:
1. Остановку приёма новых записей
2. Отправку всех оставшихся записей в очереди
3. Сохранение неотправленных записей на диск (если отправка не удалась)

## Описание полей

См. [docs/FIELDS_RU.md](docs/FIELDS_RU.md) для подробного описания всех полей.

## Примеры

- [Базовый пример (RU)](examples/ru/basic/main.go)
- [HTTP middleware (RU)](examples/ru/middleware/main.go)
- [Basic usage (EN)](examples/en/basic/main.go)
- [HTTP middleware (EN)](examples/en/middleware/main.go)

## Лицензия

MIT
