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

	"github.com/mostdoc/mostdoc/internal/attachments"
	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/db"
	"github.com/mostdoc/mostdoc/internal/directories"
	"github.com/mostdoc/mostdoc/internal/mcptokens"
	"github.com/mostdoc/mostdoc/internal/mcp"
	"github.com/mostdoc/mostdoc/internal/groups"
	"github.com/mostdoc/mostdoc/internal/pages"
	"github.com/mostdoc/mostdoc/internal/permissions"
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

	collabURL := os.Getenv("COLLAB_URL")
	if collabURL == "" {
		collabURL = "http://localhost:3002"
	}

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

	// Groups setup
	groupsRepo := groups.NewRepository(database)
	groupsHandler := groups.NewHandler(groupsRepo, logger)

	// Permissions setup
	permRepo := permissions.NewRepository(database)
	permHandler := permissions.NewHandler(permRepo, logger)

	// Repos needed for the permission resolver
	spacesRepo := spaces.NewRepository(database)
	directoriesRepo := directories.NewRepository(database)
	pagesRepo := pages.NewRepository(database)

	// Initialize FTS5 full-text search (virtual table + triggers)
	if err := pagesRepo.InitFTS5(context.Background()); err != nil {
		logger.Error("failed to init FTS5", "error", err)
		logger.Warn("FTS5 not available — ensure binary is built with -tags fts5")
	} else {
		// Backfill FTS index from existing snapshots
		if err := pagesRepo.BackfillFTS5(context.Background()); err != nil {
			logger.Error("failed to backfill FTS5", "error", err)
		} else {
			var count int
			_ = database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM page_fts").Scan(&count)
			logger.Info("FTS5 search ready", "indexed_pages", count)

			// Sample the first few rows to verify content is indexed
			if sampleRows, err := database.QueryContext(context.Background(), "SELECT page_id, substr(title,1,40), substr(content,1,100) FROM page_fts LIMIT 3"); err == nil {
				rowNum := 0
				for sampleRows.Next() {
					var samplePageID, sampleTitle, sampleContent string
					if err := sampleRows.Scan(&samplePageID, &sampleTitle, &sampleContent); err == nil {
						logger.Info("FTS5 sample",
							"page_id", samplePageID,
							"title", sampleTitle,
							"content_preview", sampleContent,
						)
						rowNum++
					}
				}
				sampleRows.Close()
				if rowNum == 0 {
					logger.Warn("FTS5 has rows but none could be sampled — possible empty content")
				}
			}
		}
	}

	// Permission resolver with callbacks for ancestor walking
	permResolver := permissions.NewResolver(
		permRepo,
		directoriesRepo.GetDirectory,          // dirGet
		spacesRepo.Get,                         // spaceGet
		pagesRepo.GetPage,                      // pageGet
		groupsRepo.ListUserGroups,              // listUG
		logger,
	)

	// Spaces setup
	spacesHandler := spaces.NewHandler(spacesRepo, logger, permResolver)

	// Directories setup
	directoriesHandler := directories.NewRESTHandler(directoriesRepo, logger, nil, permResolver)

	// SSE hub
	sseHub := sse.NewHub(logger, 256)
	sseHandler := sse.NewHandler(sseHub, logger)

	// Update directories handler with SSE hub
	directoriesHandler = directories.NewRESTHandler(directoriesRepo, logger, sseHub, permResolver)

	mux := http.NewServeMux()
	pages.RegisterRoutes(mux, pagesRepo, logger, dataDir, authService, sseHub, permResolver, collabURL)
	authHandler.RegisterRoutes(mux)
	spacesHandler.RegisterRoutes(mux)
	directoriesHandler.RegisterRESTRoutes(mux)
	groupsHandler.RegisterRoutes(mux)
	permHandler.RegisterRoutes(mux)
	attachmentsHandler := attachments.NewHandler(logger, dataDir, permResolver, authService)
	attachmentsHandler.RegisterRoutes(mux)
	sseHandler.RegisterRoutes(mux)

	// MCP tokens setup
	mcpTokenRepo := mcptokens.NewRepository(database)
	mcpTokenService := mcptokens.NewService(mcpTokenRepo)
	mcpTokenHandler := mcptokens.NewHandler(mcpTokenService, logger)
	mcpTokenHandler.RegisterRoutes(mux)

	// MCP server (MCP protocol over SSE, port 3100)
	mcpServer := mcp.NewServer(
		mcpTokenService,
		pagesRepo,
		spacesRepo,
		directoriesRepo,
		permResolver,
		logger,
	)

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

	// Start MCP SSE server on port 3100
	mcpHTTPAddr := os.Getenv("MCP_HTTP_ADDR")
	if mcpHTTPAddr == "" {
		mcpHTTPAddr = ":3100"
	}
	mcpHandler := mcpServer.Handler()
	mcpHTTPServer := &http.Server{
		Addr:    mcpHTTPAddr,
		Handler: mcpHandler,
	}
	go func() {
		logger.Info("starting MCP server (SSE)", "addr", mcpHTTPAddr)
		if err := mcpHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("mcp server error", "error", err)
		}
	}()

	// Periodic maintenance: snapshot retention cleanup (every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		// Run once at startup too
		spaces, err := spacesRepo.List(ctx)
		if err == nil {
			var retained []pages.SpaceRetention
			for _, s := range spaces {
				retained = append(retained, pages.SpaceRetention{
					ID: s.ID,
					RetentionDays: func() int64 {
						if s.SnapshotRetentionDays.Valid {
							return s.SnapshotRetentionDays.Int64
						}
						return 0
					}(),
				})
			}
			pages.RunSnapshotRetention(ctx, pagesRepo, retained, logger)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				spaces, err := spacesRepo.List(context.Background())
				if err != nil {
					logger.Error("[maintenance] list spaces", "error", err)
					continue
				}
				var retained []pages.SpaceRetention
				for _, s := range spaces {
					retained = append(retained, pages.SpaceRetention{
						ID: s.ID,
						RetentionDays: func() int64 {
							if s.SnapshotRetentionDays.Valid {
								return s.SnapshotRetentionDays.Int64
							}
							return 0
						}(),
					})
				}
				pages.RunSnapshotRetention(context.Background(), pagesRepo, retained, logger)
			}
		}
	}()

	// Periodic maintenance: Yjs snapshot compaction (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		// Run once at startup too
		pages.RunCompaction(ctx, pagesRepo, collabURL, logger, 500*1024)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pages.RunCompaction(context.Background(), pagesRepo, collabURL, logger, 500*1024)
			}
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	if err := mcpHTTPServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("mcp shutdown error", "error", err)
	}
}
