// Пример: задать дефолты сервиса/приложения/окружения один раз на клиенте.
//
// Не нужно передавать WithServiceName на каждом вызове. Укажите AppSlug,
// ServiceName, DeploymentEnv и AppVersion в Config — SDK добавит их к каждой
// записи автоматически.
package main

import (
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "homedata",     // проект / приложение
		ServiceName:   "sensorloader", // микросервис в рамках проекта
		DeploymentEnv: "PRODUCTION",
		AppVersion:    os.Getenv("BUILD_VERSION"), // напр., из CI
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// WithServiceName не нужен — appSlug/serviceName/env/version проставятся сами.
	client.CaptureError(errors.New("диск почти заполнен"), els.WithURL("/internal/disk"))
	client.CaptureMessage("запуск завершён", els.LevelInfo)

	client.Flush()
}
