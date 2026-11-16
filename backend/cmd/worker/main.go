package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
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
	var pool *pgxpool.Pool
	err = database.Retry(ctx, c.Notifications.StartupAttempts, c.Notifications.StartupBackoff, func(attempt context.Context) error {
		var connectErr error
		pool, connectErr = database.Open(attempt, c)
		if connectErr != nil {
			log.Warn("worker database startup retry", "error_class", database.FailureClass(connectErr))
		}
		return connectErr
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
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
