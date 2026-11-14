package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/modules/master"
	"wbs-golang/backend/internal/modules/notifications"
	"wbs-golang/backend/internal/modules/reports"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/httpserver"
	"wbs-golang/backend/internal/platform/logger"
)

var version = "dev"

func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(c.Env)
	gin.SetMode(gin.ReleaseMode)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, c)
	if err != nil {
		return err
	}
	defer pool.Close()
	h := health.NewHandler(health.NewService(pool, c.ReadyTimeout), log, c.AppName, c.Env, version)
	log.Info("starting HTTP server", "app", c.AppName, "version", version)
	r := httpserver.Router(c, log, h)
	authService := auth.NewService(auth.NewRepository(pool), c.SessionTTL, c.QueryTimeout)
	a := auth.NewHandler(authService, log, c.Env == "production", c.SessionTTL, c.Origins)
	a.Register(r)
	master.NewHandler(master.NewService(master.NewRepository(pool), c.QueryTimeout), a).Register(r)
	zone, err := time.LoadLocation(c.Reports.Timezone)
	if err != nil {
		return fmt.Errorf("invalid report timezone")
	}
	files, err := filestore.NewLocal(c.Reports.UploadDir, c.Reports.MaxFileBytes, c.Reports.AllowedTypes)
	if err != nil {
		return err
	}
	defer files.Close()
	reports.NewHandler(reports.NewService(reports.NewRepository(pool, zone, time.Now, reports.RandomPrefix).NotificationAttempts(c.Notifications.MaxAttempts), files, c.QueryTimeout, log), files, c.Reports, log).Register(r)
	reports.NewAdminHandler(reports.NewAdminService(reports.NewRepository(pool, zone, time.Now, reports.RandomPrefix).NotificationAttempts(c.Notifications.MaxAttempts), files, c.QueryTimeout), a, zone, c.WriteTimeout, log).Register(r)
	notifications.NewHandler(notifications.NewService(notifications.NewRepository(pool), authService, c.QueryTimeout), a, c.Notifications, ctx).Register(r)
	return httpserver.Run(ctx, httpserver.New(c, r), c.ShutdownTimeout)
}
func main() {
	if err := run(); err != nil {
		logger.New(os.Getenv("APP_ENV")).Error("startup or server failed", "error", err.Error())
		os.Exit(1)
	}
}
