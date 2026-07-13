// Package storage puts uploaded files somewhere and tells you their URL. The
// backend is chosen by config, so handlers never learn where a file lives: local
// disk today, S3/GCS behind the same interface later. Each backend lives in its
// own file (local.go).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"coachwise/src/config"
	"coachwise/src/logger"

	"github.com/google/uuid"
)

var (
	ErrTooLarge  = errors.New("file is larger than the upload limit")
	ErrMediaType = errors.New("file type is not allowed")
)

// Kind is the top-level folder a file belongs to, so no one directory grows
// without bound and a class of files can move to a CDN on its own.
type Kind string

const (
	KindMedia    Kind = "media"
	KindAvatar   Kind = "avatars"
	KindExercise Kind = "exercises"
)

type Storage interface {
	Name() string
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	// URL is absolute — the app and the seeder store URLs verbatim, so a
	// relative one breaks the moment it leaves this host.
	URL(key string) string
}

var active Storage

// Init builds the configured backend. Call once at startup in the API, the
// worker and the seeder — all three write files.
func Init() {
	switch strings.ToLower(config.Config.Storage.Provider) {
	case "", "local":
		active = newLocal()
	default:
		logger.Fatalf("storage: unknown provider %q", config.Config.Storage.Provider)
	}
	logger.Infof("storage: using %s", active.Name())
}

func Get() Storage {
	if active == nil {
		Init()
	}
	return active
}

// Key builds the object key for a new file: kind/YYYY/MM/<uuid><ext>. The name
// is random — an uploader's filename is never trusted as a path — but the
// extension carries over so the URL still says what the file is.
func Key(kind Kind, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	now := time.Now().UTC()
	return fmt.Sprintf("%s/%d/%02d/%s%s", kind, now.Year(), int(now.Month()), uuid.NewString(), ext)
}

// KeyNamed is a stable key for a file with a meaningful name (a seeded exercise
// animation keyed by slug), so re-seeding overwrites in place.
func KeyNamed(kind Kind, name string) string {
	return fmt.Sprintf("%s/%s", kind, filepath.Base(name))
}

func MaxSize() int64 {
	mb := config.Config.Storage.MaxSizeMB
	if mb <= 0 {
		mb = 10
	}
	return int64(mb) << 20
}

// CheckMediaType decides whether contentType may be uploaded. SVG is not in the
// default list on purpose: it can carry script and we serve uploads from the
// API's own origin.
func CheckMediaType(contentType string) error {
	allowed := config.Config.Storage.AllowedMIME
	if len(allowed) == 0 {
		allowed = []string{
			"image/jpeg", "image/png", "image/webp", "image/gif",
			"video/mp4", "video/webm", "video/quicktime",
		}
	}

	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	for _, a := range allowed {
		if ct == strings.ToLower(strings.TrimSpace(a)) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrMediaType, ct)
}
