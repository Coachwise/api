package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"coachwise/src/config"
	"coachwise/src/logger"
)

// local writes under a configured directory, served back by the API's static
// /uploads route.
type local struct {
	dir     string
	baseURL string
}

func newLocal() Storage {
	dir := config.Config.Storage.Dir
	if dir == "" {
		dir = "./uploads"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		logger.Fatalf("storage: bad upload dir %q: %v", dir, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		logger.Fatalf("storage: cannot create upload dir %s: %v", abs, err)
	}

	// Falls back to the URLs an existing deployment already configures.
	base := config.Config.Storage.BaseURL
	if base == "" {
		base = config.Config.MediaBaseURL
	}
	if base == "" {
		base = config.Config.PublicURL
	}

	return &local{dir: abs, baseURL: strings.TrimSuffix(base, "/")}
}

func (l *local) Name() string { return "local(" + l.dir + ")" }

func (l *local) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("storage: create dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("storage: create file: %w", err)
	}
	defer f.Close()

	// Copy no more than the limit even if the caller lied about size.
	written, err := io.Copy(f, io.LimitReader(r, MaxSize()+1))
	if err != nil {
		os.Remove(path)
		return fmt.Errorf("storage: write file: %w", err)
	}
	if written > MaxSize() {
		os.Remove(path)
		return ErrTooLarge
	}
	return nil
}

func (l *local) Delete(ctx context.Context, key string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: delete file: %w", err)
	}
	return nil
}

func (l *local) URL(key string) string {
	return l.baseURL + "/uploads/" + strings.TrimPrefix(key, "/")
}

// path resolves a key inside the upload dir, refusing anything that climbs out.
func (l *local) path(key string) (string, error) {
	clean := filepath.Clean("/" + strings.TrimPrefix(key, "/"))
	full := filepath.Join(l.dir, clean)
	if !strings.HasPrefix(full, l.dir+string(os.PathSeparator)) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return full, nil
}

// LocalDir is the directory the static /uploads route serves. Empty when the
// backend isn't local — then a CDN serves them and the API shouldn't.
func LocalDir() string {
	if l, ok := Get().(*local); ok {
		return l.dir
	}
	return ""
}
