// Пример: интеграция ELS со стандартным log/slog.
//
// Записи slog уходят в ELS. Если slog.Error содержит атрибут-ошибку
// ("err"/"error"), запись отправляется с полным стек-трейсом и текстом ошибки
// в Meta — как CaptureError. ServiceName/AppSlug из Config добавляются
// автоматически, указывать их на каждом вызове не нужно.
package main

import (
	"errors"
	"log/slog"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:      os.Getenv("ELS_API_KEY"),
		AppSlug:     "homedata",
		ServiceName: "sensorloader",
	})
	if err != nil {
		slog.Error("инициализация ELS", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	logger := slog.New(els.SlogHandler(client, &els.SlogHandlerOptions{
		AddSource:           true,
		CaptureStackOnError: true,
		// ErrorKeys по умолчанию {"err", "error"}; переопределите, если у вас
		// для ошибок используется другой ключ атрибута.
	}))
	slog.SetDefault(logger)

	slog.Info("сервис запущен", "port", 8080)
	slog.Warn("высокий процент промахов кэша", "rate", 0.42)

	if err := doWork(); err != nil {
		// Уйдёт как ошибка ELS СО стек-трейсом и Meta["error"].
		slog.Error("работа упала", "err", err, "stage", "ingest")
	}

	client.Flush()
}

func doWork() error {
	return errors.New("поток данных с датчика прерван")
}
