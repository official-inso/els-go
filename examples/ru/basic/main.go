// Пример: базовое использование ELS Go SDK.
//
// Этот пример демонстрирует инициализацию клиента, захват ошибок и сообщений,
// синхронную отправку для критичных ошибок и корректное завершение работы.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	// Инициализация клиента ELS
	client, err := els.New(els.Config{
		Endpoint:      os.Getenv("ELS_ENDPOINT"), // напр., "https://api.example.com/els"
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "my-service",
		DeploymentEnv: "PRODUCTION",
		ServiceName:   "api-gateway",

		// Опционально: настройка пакетной отправки
		BatchSize:     50,
		BatchInterval: 5 * time.Second,

		// Опционально: минимальный уровень (отбрасывает debug и info)
		MinLevel: els.LevelWarning,

		// Опционально: коллбек для отладки
		OnError: func(err error) {
			log.Printf("[ELS] внутренняя ошибка: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("Не удалось инициализировать ELS: %v", err)
	}
	// Всегда закрывайте клиент для отправки оставшихся записей
	defer client.Close()

	// --- Проверка доступности сервера ---
	ctx := context.Background()
	if err := client.Health(ctx); err != nil {
		log.Printf("ELS сервер недоступен: %v (записи будут буферизованы)", err)
	}

	// --- Захват ошибки с автоматическим stack trace (асинхронно) ---
	client.CaptureError(
		errors.New("таймаут подключения к базе данных"),
		els.WithURL("/api/users"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{
			"database": "postgres-primary",
			"timeout":  "30s",
		}),
	)

	// --- Захват предупреждения (асинхронно) ---
	client.CaptureMessage("использование памяти выше 80%", els.LevelWarning,
		els.WithURL("/health"),
		els.WithMeta(map[string]any{"memoryPct": 82.5}),
	)

	// --- Синхронная отправка для критичных ошибок (блокирует до подтверждения) ---
	err = client.SendSync(ctx, errors.New("ошибка обработки платежа"),
		els.WithURL("/api/payments/charge"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{
			"orderId": "ord_12345",
			"amount":  9900,
		}),
	)
	if err != nil {
		if els.IsRetryableErr(err) {
			log.Printf("Временная ошибка ELS: %v", err)
		} else {
			log.Printf("Постоянная ошибка ELS: %v", err)
		}
	}

	// --- Захват готовой записи (асинхронно) ---
	client.CaptureEntry(els.ErrorEntry{
		Message: "обнаружен необычный паттерн трафика",
		URL:     "/api/analytics",
		Level:   els.LevelWarning,
		Meta: map[string]any{
			"rps":       15000,
			"threshold": 10000,
		},
	})

	// Ожидание отправки всех записей
	client.Flush()
	fmt.Println("Все ошибки захвачены. Завершение работы...")
}
