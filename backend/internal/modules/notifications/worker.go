package notifications

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/platform/security"
)

type Worker struct {
	repo         *Repository
	providers    map[string]Provider
	config       config.NotificationsConfig
	log          *slog.Logger
	owner        string
	queryTimeout time.Duration
}

func NewWorker(r *Repository, email, telegram Provider, c config.NotificationsConfig, queryTimeout time.Duration, log *slog.Logger) *Worker {
	return &Worker{r, map[string]Provider{"EMAIL": email, "TELEGRAM": telegram}, c, log, security.Token()[:32], queryTimeout}
}
func (w *Worker) Run(ctx context.Context) error {
	if !w.config.WorkerEnabled {
		return nil
	}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			w.Process(ctx)
			timer.Reset(w.config.PollInterval)
		}
	}
}
func (w *Worker) Process(ctx context.Context) {
	// Claim only as many jobs as can start immediately: queued jobs must not spend their lease waiting.
	remaining := w.config.BatchSize
	for remaining > 0 && ctx.Err() == nil {
		batch := min(remaining, w.config.Concurrency)
		query, cancel := context.WithTimeout(ctx, w.queryTimeout)
		jobs, err := w.repo.Claim(query, w.owner, batch, w.config.LeaseDuration, w.config.SMTP.Enabled, w.config.Telegram.Enabled)
		cancel()
		if err != nil {
			w.log.Error("notification claim failed", "error_class", "database")
			return
		}
		if len(jobs) == 0 {
			return
		}
		var wg sync.WaitGroup
		for _, job := range jobs {
			wg.Add(1)
			go func(j Job) { defer wg.Done(); w.deliver(ctx, j) }(job)
		}
		wg.Wait()
		remaining -= len(jobs)
	}
}
func (w *Worker) deliver(ctx context.Context, j Job) {
	var m Message
	var err error
	query, cancel := context.WithTimeout(ctx, w.queryTimeout)
	if !ValidPayload(j.EventKey, j.Payload) {
		err = &Failure{Code: "PAYLOAD_INVALID"}
	} else if j.Channel == "EMAIL" {
		m, err = w.repo.EmailContent(query, j)
	} else {
		m = Message{Text: TelegramText(j.Payload), DeliveryID: j.ID}
	}
	cancel()
	var providerID string
	if err == nil {
		timeout := w.config.Telegram.Timeout
		if j.Channel == "EMAIL" {
			timeout = w.config.SMTP.Timeout
		}
		ioCtx, stop := context.WithTimeout(ctx, timeout)
		providerID, err = w.providers[j.Channel].Send(ioCtx, m)
		stop()
	}
	var failure *Failure
	delay := time.Duration(0)
	if err != nil {
		failure = Classify(err)
		delay = Backoff(j.Attempts, w.config.Retry, jitter())
		if failure.RetryAfter > delay {
			delay = failure.RetryAfter
		}
	}
	// Shutdown may interrupt provider I/O. Persist its safe outcome with a fresh bounded context.
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.queryTimeout)
	defer cancel()
	ok, finishErr := w.repo.Finish(finish, j, providerID, failure, delay)
	if finishErr != nil {
		w.log.Error("notification result persistence failed", "delivery_id", j.ID, "error_class", "database")
	} else if !ok {
		w.log.Warn("notification lease ownership lost", "delivery_id", j.ID)
	}
}
