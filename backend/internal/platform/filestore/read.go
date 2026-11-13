package filestore

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"wbs-golang/backend/internal/platform/problem"
)

type Download struct {
	Body                         io.ReadCloser
	Size                         int64
	ContentType, Extension, Name string
}
type Reader interface {
	Open(context.Context, string) (Download, error)
}

func Missing() error { return problem.New(404, "ATTACHMENT_NOT_FOUND", "Attachment tidak tersedia.") }
func (l *Local) Open(ctx context.Context, key string) (Download, error) {
	if err := ctx.Err(); err != nil {
		return Download{}, err
	}
	if !validKey.MatchString(key) {
		return Download{}, Missing()
	}
	info, err := l.root.Lstat(key)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Download{}, Missing()
	}
	file, err := l.root.Open(key)
	if err != nil {
		return Download{}, Missing()
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > l.max {
		_ = file.Close()
		return Download{}, Missing()
	}
	ext := filepath.Ext(key)
	kind := map[string]string{".jpg": "image/jpeg", ".png": "image/png", ".pdf": "application/pdf"}[ext]
	return Download{Body: file, Size: info.Size(), ContentType: kind, Extension: ext}, nil
}

// Copy bounds bytes and checks cancellation between reads; HTTP timeouts bound slow clients.
func Copy(ctx context.Context, w io.Writer, file Download) (int64, error) {
	return io.Copy(w, io.LimitReader(contextReader{ctx, file.Body}, file.Size))
}
