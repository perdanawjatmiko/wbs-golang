package filestore

import (
	"bytes"
	"context"
	"errors"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

type File struct{ Key string }
type Store interface {
	Stage(context.Context, string, string, io.Reader) (File, error)
	Publish(context.Context, File) error
	Remove(context.Context, File) error
}
type Local struct {
	root    *os.Root
	max     int64
	allowed map[string]bool
}

func NewLocal(directory string, max int64, allowed []string) (*Local, error) {
	for _, dir := range []string{directory, filepath.Join(directory, "staging")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, errors.New("private storage initialization failed")
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			return nil, errors.New("UPLOAD_DIR and staging must be private real directories (0700)")
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, errors.New("private storage unavailable")
	}
	types := map[string]bool{}
	for _, kind := range allowed {
		types[kind] = true
	}
	return &Local{root, max, types}, nil
}
func (l *Local) Close() error { return l.root.Close() }

var validKey = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|pdf)$`)

func invalid(message string) error { return problem.New(422, "INVALID_ATTACHMENT", message) }
func ValidateFilename(name string) error {
	if name == "" || len(name) > 255 || strings.ContainsAny(name, "/\\\x00") || strings.Contains(name, "..") || filepath.Base(name) != name {
		return invalid("Nama attachment tidak valid.")
	}
	return nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
func (l *Local) Stage(ctx context.Context, name, declared string, reader io.Reader) (File, error) {
	if err := ValidateFilename(name); err != nil {
		return File{}, err
	}
	ext := strings.ToLower(filepath.Ext(name))
	kind := ""
	switch ext {
	case ".jpg", ".jpeg":
		kind = "image/jpeg"
		ext = ".jpg"
	case ".png":
		kind = "image/png"
	case ".pdf":
		kind = "application/pdf"
	default:
		return File{}, invalid("Format attachment harus JPEG, PNG, atau PDF.")
	}
	if !l.allowed[kind] {
		return File{}, invalid("Tipe attachment tidak diizinkan.")
	}
	if declared != "" {
		media, _, err := mime.ParseMediaType(declared)
		if err != nil || media != kind {
			return File{}, invalid("MIME dan ekstensi attachment tidak cocok.")
		}
	}
	file := File{Key: security.Token() + ext}
	path := "staging/" + file.Key
	out, err := l.root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return File{}, err
	}
	keep := false
	defer func() {
		_ = out.Close()
		if !keep {
			_ = l.root.Remove(path)
		}
	}()
	n, err := io.Copy(out, io.LimitReader(contextReader{ctx, reader}, l.max+1))
	if err != nil {
		return File{}, err
	}
	if n > l.max {
		return File{}, problem.New(413, "ATTACHMENT_TOO_LARGE", "Ukuran attachment melebihi batas.")
	}
	if n == 0 {
		return File{}, invalid("Attachment kosong.")
	}
	if err = out.Sync(); err != nil {
		return File{}, err
	}
	if err = out.Close(); err != nil {
		return File{}, err
	}
	input, err := l.root.Open(path)
	if err != nil {
		return File{}, err
	}
	defer input.Close()
	if err = validate(ctx, input, kind, n); err != nil {
		return File{}, err
	}
	keep = true
	return file, nil
}
func validate(ctx context.Context, input *os.File, kind string, size int64) error {
	header := make([]byte, 512)
	n, err := input.Read(header)
	if err != nil && err != io.EOF {
		return err
	}
	if http.DetectContentType(header[:n]) != kind {
		return invalid("Signature attachment tidak valid.")
	}
	if _, err = input.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if kind == "application/pdf" {
		conf := model.NewStatelessConfiguration()
		conf.ValidationMode = model.ValidationStrict
		conf.Optimize = false
		conf.Limits.MaxInputBytes = size
		conf.Limits.MaxDecodeBytes = 32 << 20
		conf.Limits.MaxStreamBytes = 16 << 20
		conf.Limits.MaxObjectBytes = 16 << 20
		conf.Limits.MaxObjectCount = 5000
		conf.Limits.MaxXRefEntries = 50000
		conf.Limits.MaxRecursionDepth = 100
		conf.Limits.MaxImagePixels = 20000000
		conf.Limits.MaxImageBytes = 80 << 20
		conf.Limits.MaxObjectStreamCount = 5000
		conf.Limits.MaxObjectStreamFirst = 1 << 20
		conf.Offline = true
		conf.ValidateLinks = false
		if err = api.Validate(ctx, input, conf, nil); err != nil {
			return invalid("Struktur PDF tidak valid atau tidak dapat dibaca.")
		}
	} else {
		config, _, err := image.DecodeConfig(input)
		if err != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 20000000 {
			return invalid("Gambar tidak valid atau dimensi terlalu besar.")
		}
		if _, err = input.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, _, err = image.Decode(contextReader{ctx, input}); err != nil {
			return invalid("Isi gambar tidak valid.")
		}
		tailSize := int64(12)
		if size < tailSize {
			return invalid("Gambar terpotong.")
		}
		tail := make([]byte, tailSize)
		if _, err = input.ReadAt(tail, size-tailSize); err != nil {
			return err
		}
		if (kind == "image/jpeg" && !bytes.HasSuffix(tail, []byte{0xff, 0xd9})) || (kind == "image/png" && !bytes.Equal(tail, []byte{0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82})) {
			return invalid("Gambar terpotong atau memiliki data tambahan.")
		}
	}
	return ctx.Err()
}
func (l *Local) Publish(ctx context.Context, file File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validKey.MatchString(file.Key) {
		return errors.New("invalid storage key")
	}
	return l.root.Rename("staging/"+file.Key, file.Key)
}
func (l *Local) Remove(ctx context.Context, file File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validKey.MatchString(file.Key) {
		return errors.New("invalid storage key")
	}
	var result error
	for _, path := range []string{"staging/" + file.Key, file.Key} {
		if err := l.root.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}
	return result
}
