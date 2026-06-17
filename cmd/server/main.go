package main

import (
	"avito-easy-report/internal/config"
	"avito-easy-report/internal/database"
	"avito-easy-report/internal/handler"
	"avito-easy-report/internal/middleware"
	"avito-easy-report/internal/service"
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Инициализация БД
	if err := database.Init(cfg); err != nil {
		log.Fatal("Ошибка БД:", err)
	}
	if err := database.Migrate(&database.User{}); err != nil {
		log.Fatal("Ошибка миграции:", err)
	}
	if err := database.SeedAdmin(); err != nil {
		log.Fatal("Ошибка seed admin:", err)
	}

	service.InitAuth(cfg)

	store := service.NewReportStore()
	cabinetStore := service.NewCabinetStore()
	reportHandler := handler.NewHandler(store)
	authHandler := handler.NewAuthHandler()
	cabinetHandler := handler.NewCabinetHandler(cabinetStore, store)

	r := setupRouter(cfg, reportHandler, authHandler, cabinetHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Сервер запущен на http://localhost:%s", cfg.Port)
		log.Printf("Учётка по умолчанию: admin / admin")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Ошибка сервера:", err)
		}
	}()

	<-ctx.Done()
	log.Println("Завершение работы...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Ошибка при остановке:", err)
	}
	log.Println("Сервер остановлен")
}

func setupRouter(cfg *config.Config, reportHandler *handler.Handler, authHandler *handler.AuthHandler, cabHandler *handler.CabinetHandler) *gin.Engine {
	r := gin.Default()

	// CORS
	if cfg.CORSOrigin != "" {
		r.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", cfg.CORSOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
		})
	}

	// Health-check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "avito-easy-report"})
	})

	// Публичные эндпоинты
	r.POST("/api/auth/login", authHandler.Login)
	r.POST("/api/auth/register", authHandler.Register)

	// Защищённые эндпоинты
	protected := r.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		protected.GET("/auth/me", authHandler.Me)

		protected.POST("/upload", reportHandler.UploadReport)
		protected.GET("/reports", reportHandler.ListReports)
		protected.GET("/reports/multi", reportHandler.MultiStats)
		protected.GET("/reports/compare", reportHandler.CompareReports)
		protected.GET("/reports/:id/stats", reportHandler.GetStats)

		// Кабинеты
		protected.GET("/cabinets", cabHandler.List)
		protected.POST("/cabinets", cabHandler.Create)
		protected.DELETE("/cabinets/:id", cabHandler.Delete)
		protected.GET("/cabinets/:id/reports", cabHandler.ListReports)
		protected.DELETE("/reports/:id", reportHandler.DeleteReport)
		protected.GET("/export", reportHandler.ExportAll)

		admin := protected.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", authHandler.ListUsers)
			admin.PUT("/users/:id/role", authHandler.UpdateUserRole)
			admin.DELETE("/users/:id", authHandler.DeleteUser)
		}
	}

	// SPA fallback — только в production
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "эндпоинт не найден"})
			return
		}
		c.File("./frontend/dist/index.html")
	})

	return r
}
