package main

import (
	"net/http"
	"os"

	"tip2_pr7/services/tasks/internal/client/authclient"
	httpapi "tip2_pr7/services/tasks/internal/http"
	"tip2_pr7/services/tasks/internal/service"
	"tip2_pr7/services/tasks/internal/storage/postgres"
	sharedlogger "tip2_pr7/shared/logger"
	"tip2_pr7/shared/metrics"
	"tip2_pr7/shared/middleware"

	"go.uber.org/zap"
)

func main() {
	port := getEnv("TASKS_PORT", "8082")
	authGRPCAddr := getEnv("AUTH_GRPC_ADDR", "localhost:50051")
	dbDSN := getEnv("TASKS_DB_DSN", "postgres://tasks:tasks@localhost:5432/tasksdb?sslmode=disable")

	logger, err := sharedlogger.New("tasks")
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	db, err := postgres.Open(dbDSN)
	if err != nil {
		logger.Fatal("open postgres failed", zap.Error(err), zap.String("component", "postgres"))
	}
	defer func() { _ = db.Close() }()

	taskRepo := postgres.New(db)
	taskService := service.New(taskRepo)

	authClient, err := authclient.New(authGRPCAddr, logger)
	if err != nil {
		logger.Fatal("create auth client failed", zap.Error(err), zap.String("component", "auth_client"))
	}
	defer func(authClient *authclient.Client) {
		err := authClient.Close()
		if err != nil {
			panic(err)
		}
	}(authClient)

	handler := httpapi.New(taskService, authClient, logger)

	mux := http.NewServeMux()
	handler.Register(mux)
	mux.Handle("GET /metrics", metrics.Handler())

	app := middleware.RequestID(
		middleware.SecurityHeaders(
			metrics.InstrumentHTTP(
				middleware.RequireDoubleSubmitCSRF(
					middleware.AccessLog(logger)(mux),
				),
			),
		),
	)

	addr := ":" + port
	logger.Info(
		"tasks service starting",
		zap.String("address", addr),
		zap.String("auth_grpc_addr", authGRPCAddr),
	)

	if err := http.ListenAndServe(addr, app); err != nil {
		logger.Fatal("tasks service failed", zap.Error(err), zap.String("component", "http_server"))
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
