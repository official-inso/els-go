# Поля записи ошибки ELS

Все поля, отправляемые Go SDK в ELS API.

## Обязательные

| Поле | Тип | Макс. длина | Описание |
|------|-----|-------------|----------|
| `message` | string | 10 000 | Текст ошибки |
| `url` | string | 2 000 | URL, где произошла ошибка. Используйте `WithURL()` или `WithRequest(r)` |

## Автоматически заполняемые

SDK заполняет их сам — не нужно устанавливать вручную.

| Поле | Источник | Описание |
|------|----------|----------|
| `timestamp` | `time.Now().UTC()` | ISO 8601 (RFC3339Nano) |
| `level` | `Config.DefaultLevel` | Серьёзность: critical/error/warning/info/debug |
| `source` | `Config.DefaultSource` | Источник: server/client |
| `appSlug` | `Config.AppSlug` | Идентификатор приложения |
| `deploymentEnv` | `Config.DeploymentEnv` | Нормализуется сервером (dev→DEV, prod→PRODUCTION) |
| `serviceName` | `Config.ServiceName` | Название микросервиса |
| `sessionId` | Автогенерация | ID сессии процесса для корреляции |
| `stack` | `runtime.Callers` | Stack trace (только для `CaptureError`) |

## Опциональные

Устанавливаются через опции (`WithX()`) или напрямую в `ErrorEntry`:

| Поле | Тип | Макс. | Опция | Описание |
|------|-----|-------|-------|----------|
| `stack` | string | 50 000 | `WithStack(s)` | Переопределить авто-stack |
| `componentStack` | string | 50 000 | `WithComponentStack(s)` | Трейс компонентов фреймворка |
| `userAgent` | string | 1 000 | `WithUserAgent(ua)` | User-Agent клиента |
| `language` | string | 20 | `WithLanguage(l)` | Локаль (напр., "ru-RU") |
| `screenSize` | string | 20 | — | Экран "WxH" (только клиент) |
| `viewportSize` | string | 20 | — | Вьюпорт "WxH" (только клиент) |
| `referrer` | string | 2 000 | `WithReferrer(r)` | HTTP Referer |
| `meta` | object | — | `WithMeta(m)` | Произвольные данные |

## Удобные опции

| Опция | Что делает |
|-------|-----------|
| `WithRequest(r *http.Request)` | Извлекает URL, UserAgent, Referrer, Language + добавляет `http.method`, `http.host`, `http.remoteAddr`, `http.requestId` в Meta |
| `WithCause(err)` | Обходит цепочку `Unwrap()`, сохраняет причины в `meta["error.causes"]` |

## Значения Level

| Значение | Константа | Когда использовать |
|----------|-----------|-------------------|
| `critical` | `els.LevelCritical` | Падение системы, потеря данных |
| `error` | `els.LevelError` | Операция провалилась |
| `warning` | `els.LevelWarning` | Потенциальная проблема |
| `info` | `els.LevelInfo` | Значимое событие |
| `debug` | `els.LevelDebug` | Диагностика |

## Значения Source

| Значение | Константа | Описание |
|----------|-----------|----------|
| `server` | `els.SourceServer` | Ошибка бэкенда |
| `client` | `els.SourceClient` | Ошибка фронтенда |

## Нормализация окружения

Сервер нормализует `deploymentEnv` без учёта регистра:

| Отправляете | Хранится как |
|-------------|--------------|
| dev, development, test | `DEV` |
| staging, stage, stg | `STAGING` |
| prod, production | `PRODUCTION` |
| остальное | UPPERCASE |

## Поля, генерируемые сервером

Эти поля вычисляются на стороне сервера (нельзя установить клиентом):

| Поле | Описание |
|------|----------|
| `traceId` | Уникальный ID (формат: `SRV-<timestamp>-<random>`) |
| `browser` | Определяется из userAgent |
| `urlPath` | Нормализованный путь (UUID → `:id`) |
| `errorCategory` | Автокатегоризация из message |
| `fingerprint` | Хеш от message + stack + source |
| `ip` | IP клиента из запроса |

## Поля контекста пользователя

При вызове `SetUser()` эти поля автоматически добавляются в Meta:

| Ключ в Meta | Источник |
|-------------|----------|
| `user.id` | `UserContext.ID` |
| `user.email` | `UserContext.Email` |
| `user.name` | `UserContext.Name` |
| `user.<key>` | `UserContext.Extra[key]` |
