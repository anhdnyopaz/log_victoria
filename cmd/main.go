package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anhdnyopaz/go_victorialog/internal/logger"
	"github.com/anhdnyopaz/go_victorialog/internal/service"
	"github.com/gorilla/mux"
)

func StartVictoriaLogService() (*logger.VictoriaLogsLogger, func(), error) {
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
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}

	cleanup := func() {
		if err := vlLogger.Close(); err != nil {
			log.Printf("Error closing logger: %v", err)
		}
	}

	return vlLogger, cleanup, nil
}

func main() {
	fmt.Println("Starting server...")

	vlLogger, cleanup, err := StartVictoriaLogService()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()
	
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
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			vlLogger.Fatal(context.Background(), "Failed to start server", map[string]interface{}{
				"error": err.Error(),
			})
		}

	}()

	//go demoLogs(vlLogger)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	vlLogger.Info(context.Background(), "Shutting down server", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		vlLogger.Error(context.Background(), "Failed to shutdown server", map[string]interface{}{
			"error": err.Error(),
		})
	}

}

func demoLogs(logger logger.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++
			ctx := context.WithValue(context.Background(), "trace_id", fmt.Sprintf("demo_trace_%d", counter))

			// Different log levels
			logger.Debug(ctx, "Debug message for testing", map[string]interface{}{
				"counter": counter,
				"type":    "debug_demo",
			})

			logger.Info(ctx, "Processing demo data", map[string]interface{}{
				"counter":    counter,
				"batch_size": 100,
				"type":       "info_demo",
			})

			if counter%5 == 0 {
				logger.Warn(ctx, "This is a warning message", map[string]interface{}{
					"counter": counter,
					"type":    "warn_demo",
				})
			}

			if counter%10 == 0 {
				logger.Error(ctx, "Simulated error occurred", map[string]interface{}{
					"counter":    counter,
					"error_code": "DEMO_ERROR",
					"type":       "error_demo",
				})
			}

			if counter >= 100 {
				return
			}

		}
	}

}

func traceMiddleware(logger logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceId := fmt.Sprintf("trace_%d", time.Now().UnixNano())
			ctx := context.WithValue(r.Context(), "trace_id", traceId)

			logger.Info(ctx, "Request received", map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"trace_id":   traceId,
				"user_agent": r.UserAgent(),
				"remote_ip":  r.RemoteAddr,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getUserHandler(userService *service.UserService, vlLogger *logger.VictoriaLogsLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userId := vars["id"]

		user, err := userService.GetUser(r.Context(), userId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(fmt.Sprintf(`{"id":"%s", "username":"%s","email":"%s"`,
			userId, user.Username, user.Email)))
		if err != nil {
			return
		}
	}

}

func createUserHandler(userService *service.UserService, vlLogger *logger.VictoriaLogsLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := service.User{
			ID:       fmt.Sprintf("user_%d", time.Now().Unix()),
			Username: r.URL.Query().Get("username"),
			Email:    r.URL.Query().Get("email"),
		}
		if err := userService.CreateUser(r.Context(), user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("User created successfully!"))
		if err != nil {
			return
		}

	}
}

func healthHandler(logger logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug(r.Context(), "Health check requested", nil)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			return
		}
	}
}

func getEnv(env string, fallback string) string {
	if value := os.Getenv(env); value != "" {
		return value
	}
	return fallback
}
