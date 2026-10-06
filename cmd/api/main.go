package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"
	"uptime/internal/api/handlers"
	"uptime/internal/api/middleware"
	"uptime/internal/auth"
	"uptime/internal/realtime"
	"uptime/internal/storage/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	jwtSvc := auth.NewJWTService(os.Getenv("JWT_SECRET"))
	
	// Initialize Repositories
	monitorRepo := postgres.NewMonitorRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	notifRepo := postgres.NewNotificationRepository(pool)
	incidentRepo := postgres.NewIncidentRepository(pool)
	statusPageRepo := postgres.NewStatusPageRepository(pool)

	// Initialize Services
	authSvc := auth.NewAuthService(userRepo, jwtSvc)
	hub := realtime.NewHub(logger, rdb)
	go hub.Run()

	// Handlers
	authHandler := handlers.NewAuthHandler(authSvc)
	monitorHandler := handlers.NewMonitorHandler(monitorRepo)
	statusPageHandler := handlers.NewStatusPageHandler(statusPageRepo)
	notifHandler := handlers.NewNotificationHandler(notifRepo)
	incidentHandler := handlers.NewIncidentHandler(incidentRepo)

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/status/{slug}", statusPageHandler.GetPublic)

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /api/monitors", monitorHandler.List)
	protectedMux.HandleFunc("POST /api/monitors", monitorHandler.Create)
	protectedMux.HandleFunc("GET /api/monitors/{id}/stats", monitorHandler.Stats)
	protectedMux.HandleFunc("GET /api/notifications", notifHandler.List)
	protectedMux.HandleFunc("POST /api/notifications", notifHandler.Create)
	protectedMux.HandleFunc("DELETE /api/notifications/", notifHandler.Delete)
	protectedMux.HandleFunc("GET /api/incidents", incidentHandler.List)
	protectedMux.HandleFunc("GET /api/status-pages", statusPageHandler.List)
	protectedMux.HandleFunc("POST /api/status-pages", statusPageHandler.Create)
	protectedMux.HandleFunc("DELETE /api/status-pages/", statusPageHandler.Delete)
	
	mux.Handle("/api/", jwtSvc.AuthMiddleware(protectedMux))
	mux.Handle("/ws", hub)

	// CORS Setup
	origins := os.Getenv("ALLOWED_ORIGINS")
	var allowedOrigins []string
	if origins != "" {
		parts := strings.Split(origins, ",")
		for _, p := range parts {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(p))
		}
	} else {
		allowedOrigins = []string{"*"}
	}
	corsMiddleware := middleware.CORS(allowedOrigins)

	server := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(mux),
	}

	go func() {
		logger.Info("starting api server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
}
