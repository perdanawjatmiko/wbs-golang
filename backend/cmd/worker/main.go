package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/notifications"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/logger"
	"wbs-golang/backend/internal/platform/mailer"
	"wbs-golang/backend/internal/platform/telegram"
)

func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(c.Env)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if !c.Notifications.WorkerEnabled {
		log.Info("notification worker disabled")
		return nil
	}
	pool, err := database.Open(ctx, c)
	if err != nil {
		return err
	}
	defer pool.Close()
	return notifications.NewWorker(notifications.NewRepository(pool), mailer.New(c.Notifications.SMTP), telegram.New(c.Notifications.Telegram), c.Notifications, c.QueryTimeout, log).Run(ctx)
}
func main() {
	if err := run(); err != nil {
		logger.New(os.Getenv("APP_ENV")).Error("worker startup failed", "error", err.Error())
		os.Exit(1)
	}
}
