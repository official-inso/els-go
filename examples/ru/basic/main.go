// Пример: базовое использование ELS Go SDK.
//
// Этот пример демонстрирует инициализацию клиента, захват ошибок
// и сообщений, а также корректное завершение работы.
package main

import (
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

	// --- Захват ошибки с автоматическим stack trace ---
	client.CaptureError(
		errors.New("таймаут подключения к базе данных"),
		els.WithURL("/api/users"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{
			"database": "postgres-primary",
			"timeout":  "30s",
		}),
	)

	// --- Захват информационного сообщения ---
	client.CaptureMessage("сервис успешно запущен", els.LevelInfo,
		els.WithURL("/"),
	)

	// --- Захват предупреждения ---
	client.CaptureMessage("использование памяти выше 80%", els.LevelWarning,
		els.WithURL("/health"),
		els.WithMeta(map[string]any{"memoryPct": 82.5}),
	)

	// --- Захват готовой записи ---
	client.CaptureEntry(els.ErrorEntry{
		Message: "ошибка обработки платежа",
		URL:     "/api/payments/charge",
		Level:   els.LevelError,
		Meta: map[string]any{
			"orderId": "ord_12345",
			"amount":  9900,
		},
	})

	// Даём фоновому воркеру время на отправку
	time.Sleep(time.Second)
	fmt.Println("Все ошибки захвачены. Завершение работы...")
}
