// Пример: использование HTTP middleware с ELS Go SDK.
//
// Этот пример демонстрирует встроенный middleware для автоматического
// перехвата паник в HTTP-хендлерах.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	// Инициализация клиента ELS
	client, err := els.New(els.Config{
		Endpoint:      os.Getenv("ELS_ENDPOINT"),
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "web-api",
		DeploymentEnv: "PRODUCTION",
		ServiceName:   "user-service",
		BatchInterval: 2 * time.Second,
		OnError: func(err error) {
			log.Printf("[ELS] %v", err)
		},
	})
	if err != nil {
		log.Fatalf("Не удалось инициализировать ELS: %v", err)
	}
	defer client.Close()

	// Настройка HTTP-маршрутов
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// Имитация ошибки в бизнес-логике
		userID := r.URL.Query().Get("id")
		if userID == "" {
			// Ручной захват нефатальной ошибки
			client.CaptureMessage("отсутствует ID пользователя в запросе", els.LevelWarning,
				els.WithURL(r.URL.String()),
				els.WithUserAgent(r.UserAgent()),
				els.WithReferrer(r.Referer()),
			)
			http.Error(w, "требуется параметр id", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": userID, "name": "Иван"})
	})

	mux.HandleFunc("/api/panic", func(w http.ResponseWriter, r *http.Request) {
		// Эта паника будет автоматически перехвачена middleware
		panic("неожиданный nil pointer при поиске пользователя")
	})

	// Оборачиваем mux middleware для перехвата паник.
	// Любая паника в хендлерах будет захвачена как CRITICAL ошибка.
	handler := client.Middleware(mux)

	// Добавляем recovery-обработчик сверху, чтобы сервер не падал
	safeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			}
		}()
		handler.ServeHTTP(w, r)
	})

	addr := ":8080"
	fmt.Printf("Сервер слушает на %s\n", addr)
	fmt.Println("Попробуйте: GET /api/health, GET /api/users?id=1, GET /api/panic")
	log.Fatal(http.ListenAndServe(addr, safeHandler))
}
