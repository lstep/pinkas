package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/db"
	"github.com/mostdoc/mostdoc/internal/directories"
	"github.com/mostdoc/mostdoc/internal/pages"
	"github.com/mostdoc/mostdoc/internal/sse"
	"github.com/mostdoc/mostdoc/internal/spaces"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	socketPath := os.Getenv("SOCKET_PATH")

	database, err := db.Open(dataDir)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://migrations"
	}
	if err := db.Migrate(database, migrationsPath); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	if err := db.Seed(database); err != nil {
		logger.Error("failed to seed database", "error", err)
		os.Exit(1)
	}

	// Auth setup
	authRepo := auth.NewRepository(database)
	authService, err := auth.NewService(authRepo, dataDir)
	if err != nil {
		logger.Error("failed to create auth service", "error", err)
		os.Exit(1)
	}
	authHandler := auth.NewHandler(authService, authRepo, logger)

	// Spaces setup
	spacesRepo := spaces.NewRepository(database)
	spacesHandler := spaces.NewHandler(spacesRepo, logger)

	// Directories setup
	directoriesRepo := directories.NewRepository(database)
	directoriesHandler := directories.NewRESTHandler(directoriesRepo, logger, nil)

	// SSE hub
	sseHub := sse.NewHub(logger, 256)
	sseHandler := sse.NewHandler(sseHub, logger)

	// Update directories handler with SSE hub
	directoriesHandler = directories.NewRESTHandler(directoriesRepo, logger, sseHub)

	mux := http.NewServeMux()
	pagesRepo := pages.NewRepository(database)
	pages.RegisterRoutes(mux, pagesRepo, logger, dataDir, authService, sseHub)
	authHandler.RegisterRoutes(mux)
	spacesHandler.RegisterRoutes(mux)
	directoriesHandler.RegisterRESTRoutes(mux)
	sseHandler.RegisterRoutes(mux)

	// Wrap with auth middleware
	wrapped := auth.Middleware(authService)(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      wrapped,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		var err error
		if socketPath != "" {
			os.Remove(socketPath)
			listener, listenErr := net.Listen("unix", socketPath)
			if listenErr != nil {
				logger.Error("failed to create unix socket listener", "error", listenErr)
				os.Exit(1)
			}
			defer os.Remove(socketPath)
			logger.Info("starting Go API (unix socket)", "socket", socketPath)
			err = server.Serve(listener)
		} else {
			logger.Info("starting Go API (tcp)", "port", port)
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
