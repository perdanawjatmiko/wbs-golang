package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/problem"
)

func SSEFrame(e RealtimeEvent) string {
	data, _ := json.Marshal(e)
	return fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", e.Sequence, e.Name, data)
}
func (h *Handler) stream(c *gin.Context) {
	cursor, err := parseCursor(c.GetHeader("Last-Event-ID"))
	if err != nil {
		h.a.Fail(c, err)
		return
	}
	if h.connections.Add(1) > int64(h.config.SSEMaxConnections) {
		h.connections.Add(-1)
		h.a.Fail(c, problem.New(503, "SSE_CAPACITY", "Koneksi realtime penuh. Coba lagi nanti."))
		return
	}
	defer h.connections.Add(-1)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	stop := context.AfterFunc(h.shutdown, cancel)
	defer stop()
	token, _ := c.Cookie(auth.CookieName)
	// Start live streams at the committed high-water mark; inbox remains the durable full list.
	if c.GetHeader("Last-Event-ID") == "" {
		query, done := context.WithTimeout(ctx, h.s.timeout)
		cursor, err = h.s.repo.Cursor(query)
		done()
		if err != nil {
			h.a.Fail(c, err)
			return
		}
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store, private")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Connection", "keep-alive")
	controller := http.NewResponseController(c.Writer)
	write := func(frame string) bool {
		if err := controller.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil && err != http.ErrNotSupported {
			return false
		}
		if _, err := fmt.Fprint(c.Writer, frame); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	if !write(fmt.Sprintf("retry: 3000\nid: %d\n: connected\n\n", cursor)) {
		return
	}
	poll := time.NewTicker(h.config.SSEPoll)
	defer poll.Stop()
	heart := time.NewTicker(h.config.SSEHeartbeat)
	defer heart.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-poll.C:
			// Every batch checks the live session/RBAC before any report metadata is disclosed.
			query, done := context.WithTimeout(ctx, h.s.timeout)
			p, err := h.s.Authorize(query, token, "report.view")
			if err != nil {
				done()
				write("event: access.revoked\ndata: {}\n\n")
				return
			}
			events, err := h.s.repo.Replay(query, p.ID, cursor, h.config.SSEReplay+1)
			done()
			if err != nil {
				return
			}
			if len(events) > h.config.SSEReplay { // Bound replay and ask UI to refresh its durable inbox.
				query, done := context.WithTimeout(ctx, h.s.timeout)
				cursor, err = h.s.repo.Cursor(query)
				done()
				if err != nil {
					return
				}
				if !write(fmt.Sprintf("id: %d\nevent: inbox.reset\ndata: {}\n\n", cursor)) {
					return
				}
				continue
			}
			for _, e := range events {
				if !write(SSEFrame(e)) {
					return
				}
				cursor = e.Sequence
			}
		case <-heart.C:
			if !write(": heartbeat\n\n") {
				return
			}
		}
	}
}
