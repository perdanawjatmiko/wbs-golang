package filestore

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"wbs-golang/backend/internal/testutil"
)

func TestValidFilesAndPrivateStorage(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	store, err := NewLocal(directory, 5<<20, []string{"image/jpeg", "image/png", "application/pdf"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, tt := range []struct {
		name, kind string
		data       []byte
	}{{"proof.jpeg", "image/jpeg", testutil.JPEG()}, {"proof.png", "image/png", testutil.PNG()}, {"proof.pdf", "application/pdf", testutil.PDF()}} {
		t.Run(tt.kind, func(t *testing.T) {
			f, err := store.Stage(context.Background(), tt.name, tt.kind, bytes.NewReader(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(f.Key, "proof") || !validKey.MatchString(f.Key) {
				t.Fatal("unsafe key")
			}
			if _, err = os.Stat(filepath.Join(directory, f.Key)); !os.IsNotExist(err) {
				t.Fatal("published before commit hook")
			}
			if err = store.Publish(context.Background(), f); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(filepath.Join(directory, f.Key))
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("unsafe file permissions", err)
			}
			if err = store.Remove(context.Background(), f); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestInvalidFilesAndCleanup(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	store, err := NewLocal(directory, 1024, []string{"image/jpeg", "image/png", "application/pdf"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, tt := range []struct {
		name, kind string
		data       []byte
	}{{"empty.png", "image/png", nil}, {"evil.png", "image/png", []byte("<script>alert(1)</script>")}, {"wrong.pdf", "application/pdf", testutil.PNG()}, {"wrong.png", "image/jpeg", testutil.PNG()}, {"large.png", "image/png", bytes.Repeat([]byte{0}, 1025)}, {"../escape.png", "image/png", testutil.PNG()}, {`..\escape.png`, "image/png", testutil.PNG()}, {"bad.pdf", "application/pdf", []byte("%PDF-1.7\ninvalid\n%%EOF")}, {"broken.png", "image/png", testutil.PNG()[:40]}, {"suffix.png", "image/png", append(testutil.PNG(), []byte("<html>")...)}} {
		if _, err := store.Stage(context.Background(), tt.name, tt.kind, bytes.NewReader(tt.data)); err == nil {
			t.Fatal("invalid accepted", tt.name)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(directory, "staging"))
	if len(entries) != 0 {
		t.Fatal("invalid files leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = store.Stage(ctx, "canceled.png", "image/png", bytes.NewReader(testutil.PNG())); err == nil {
		t.Fatal("cancellation ignored")
	}
}
func TestStorageFailure(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	store, err := NewLocal(directory, 1024, []string{"image/png"})
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	if _, err = store.Stage(context.Background(), "proof.png", "image/png", bytes.NewReader(testutil.PNG())); err == nil {
		t.Fatal("closed storage accepted upload")
	}
}
