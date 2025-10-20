package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anhdnyopaz/go_victorialog/internal/logger"
	"github.com/anhdnyopaz/go_victorialog/internal/service"
	"github.com/gorilla/mux"
)

func main() {
	config := &logger.Config{
		VictoriaLogsURL: getEnv("VICTORIA_LOGS_URL", "http://localhost:9428/insert/jsonline"),
		ServiceName:     "demo-api",
		BatchSize:       50,
		FlushInterval:   3 * time.Second,
		MaxRetries:      3,
		Timeout:         5 * time.Second,
		BufferSize:      500,
		Async:           true,
	}
	vlLogger, err := logger.NewVictoriaLogsLogger(config)
	if err != nil {
		log.Fatal("Failed to create logger", err)
	}

	defer vlLogger.Close()
	// Init Services
	userService := service.NewUserService(vlLogger)

	router := mux.NewRouter()

	router.HandleFunc("/health", healthHandler(vlLogger)).Methods("GET")

	router.HandleFunc("/users", createUserHandler(userService, vlLogger)).Methods("POST")

	router.HandleFunc("/users/{id}", getUserHandler(userService, vlLogger)).Methods("GET")

	router.Use(traceMiddleware(vlLogger))
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,

		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		vlLogger.Info(context.Background(), "Starting server", map[string]interface{}{
			"port": getEnv("PORT", "8080"),
		})

	}()

}

func traceMiddleware(vlLogger *logger.VictoriaLogsLogger) mux.MiddlewareFunc {

}

func getUserHandler(userService *service.UserService, vlLogger *logger.VictoriaLogsLogger) func(http.ResponseWriter, *http.Request) {

}

func createUserHandler(userService *service.UserService, vlLogger *logger.VictoriaLogsLogger) func(http.ResponseWriter, *http.Request) {

}

func healthHandler(vlLogger *logger.VictoriaLogsLogger) func(http.ResponseWriter, *http.Request) {

}

func getEnv(env string, fallback string) string {
	value := os.Getenv(env)
	if value == "" {
		return fallback
	}
	return value
}
