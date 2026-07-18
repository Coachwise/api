package main

import (
	"coachwise/src/storage"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
)

// System climbing exercise library seeder. It is DATA-DRIVEN: the content lives
// under <seedDataDir>/exercises/ (outside this repo), not compiled in, so the
// library can grow by dropping files. Layout (see that dir's README):
//
//	exercises/
//	  categories.json      — [{slug, sport_type, sort, name:{en,fa}}, ...]
//	  items/<slug>.json     — one exercise per file
//	  media/<slug>.svg      — its animation
//
// Idempotent: categories and exercises are keyed by slug and upserted, so
// re-running never duplicates. Each exercise's SVG is copied into the served
// uploads dir (uploads/exercises/<slug>.svg) and linked via a media row.

type i18nMap map[string]string

type categoryFile struct {
	Slug      string  `json:"slug"`
	SportType string  `json:"sport_type"`
	Sort      int     `json:"sort"`
	Name      i18nMap `json:"name"`
}

type exerciseFile struct {
	Slug        string    `json:"slug"`
	Category    string    `json:"category"`
	SportType   string    `json:"sport_type"`
	Name        i18nMap   `json:"name"`
	Description i18nMap   `json:"description"`
	Media       string    `json:"media"` // filename under media/
	Sets        []setSpec `json:"sets"`
}

// setSpec is a compact protocol entry; `count` expands into that many identical
// sets. Exactly one of reps/seconds is used (duration takes precedence).
type setSpec struct {
	Count       int    `json:"count"`
	Reps        int    `json:"reps"`
	Seconds     int    `json:"seconds"`
	RestSeconds int    `json:"rest_seconds"`
	Name        string `json:"name"`
}

const nsPerSec int64 = 1_000_000_000

func seedExercises(db *sql.DB) error {
	base := filepath.Join(seedDataDir, "exercises")
	if _, err := os.Stat(base); err != nil {
		return fmt.Errorf("seed-data exercises dir %q not found (use -dir): %w", base, err)
	}

	// 1) Categories.
	var cats []categoryFile
	if err := readJSON(filepath.Join(base, "categories.json"), &cats); err != nil {
		return fmt.Errorf("categories.json: %w", err)
	}
	catID := map[string]string{}
	for _, c := range cats {
		sport := defaultSport(c.SportType)
		name, _ := json.Marshal(c.Name)
		var id string
		if err := db.QueryRow(`
			INSERT INTO exercise_categories (slug, name_i18n, sport_type, sort_order)
			VALUES ($1, $2::jsonb, $3::exercise_sport_type, $4)
			ON CONFLICT (slug) DO UPDATE
			  SET name_i18n = EXCLUDED.name_i18n,
			      sport_type = EXCLUDED.sport_type,
			      sort_order = EXCLUDED.sort_order,
			      updated_at = now()
			RETURNING id`, c.Slug, string(name), sport, c.Sort).Scan(&id); err != nil {
			return fmt.Errorf("category %s: %w", c.Slug, err)
		}
		catID[c.Slug] = id
	}

	// 2) Exercises, one file each.
	files, err := filepath.Glob(filepath.Join(base, "items", "*.json"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	count := 0
	for _, f := range files {
		var e exerciseFile
		if err := readJSON(f, &e); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		cid, ok := catID[e.Category]
		if !ok {
			return fmt.Errorf("exercise %s: unknown category %q", e.Slug, e.Category)
		}

		// Store the animation through the storage service, link a media row. The
		// extension is preserved, so switching a slug from .svg to .mp4/.webp/.gif
		// later needs no code change — just drop the new file + reseed.
		var mediaID any
		if e.Media != "" {
			ext := filepath.Ext(e.Media)
			if ext == "" {
				ext = ".svg"
			}
			name := e.Slug + ext
			url, err := putMedia(filepath.Join(base, "media", e.Media), name)
			if err != nil {
				return fmt.Errorf("media %s: %w", e.Slug, err)
			}
			var id string
			if err := db.QueryRow(`
				INSERT INTO media (url, filename)
				VALUES ($1, $2)
				ON CONFLICT (url) DO UPDATE SET filename = EXCLUDED.filename
				RETURNING id`, url, name).Scan(&id); err != nil {
				return fmt.Errorf("media row %s: %w", e.Slug, err)
			}
			mediaID = id
		}

		nameI18n, _ := json.Marshal(e.Name)
		descI18n, _ := json.Marshal(e.Description)

		var exID string
		if err := db.QueryRow(`
			INSERT INTO exercises
			  (user_id, slug, name, description, name_i18n, description_i18n,
			   public, sport_type, category_id, media_id,
			   track_weight, track_distance, track_grade, track_height)
			VALUES
			  (NULL, $1, $2, $3, $4::jsonb, $5::jsonb,
			   true, $6::exercise_sport_type, $7, $8,
			   -- Climbing is graded (grade + wall height), cardio is distance, the
			   -- rest is weight — matching the metrics migration's backfill.
			   $6 NOT IN ('CLIMBING', 'CARDIO'),
			   $6 = 'CARDIO',
			   $6 = 'CLIMBING',
			   $6 = 'CLIMBING')
			ON CONFLICT (slug) DO UPDATE SET
			  name = EXCLUDED.name,
			  description = EXCLUDED.description,
			  name_i18n = EXCLUDED.name_i18n,
			  description_i18n = EXCLUDED.description_i18n,
			  public = EXCLUDED.public,
			  sport_type = EXCLUDED.sport_type,
			  category_id = EXCLUDED.category_id,
			  media_id = EXCLUDED.media_id,
			  track_weight = EXCLUDED.track_weight,
			  track_distance = EXCLUDED.track_distance,
			  track_grade = EXCLUDED.track_grade,
			  track_height = EXCLUDED.track_height,
			  updated_at = now()
			RETURNING id`,
			e.Slug, e.Name["en"], e.Description["en"], string(nameI18n), string(descI18n),
			defaultSport(e.SportType), cid, mediaID).Scan(&exID); err != nil {
			return fmt.Errorf("exercise %s: %w", e.Slug, err)
		}

		// Rebuild sets idempotently from the protocol specs.
		if _, err := db.Exec(`DELETE FROM sets WHERE exercise_id = $1`, exID); err != nil {
			return fmt.Errorf("sets clear %s: %w", e.Slug, err)
		}
		setNum := 0
		for _, s := range e.Sets {
			cnt := s.Count
			if cnt < 1 {
				cnt = 1
			}
			restNs := int64(s.RestSeconds) * nsPerSec
			var name any
			if s.Name != "" {
				name = s.Name
			}
			for i := 0; i < cnt; i++ {
				setNum++
				if s.Seconds > 0 {
					if _, err := db.Exec(`INSERT INTO sets (exercise_id, name, set_number, rest_time, duration) VALUES ($1,$2,$3,$4,$5)`,
						exID, name, setNum, restNs, int64(s.Seconds)*nsPerSec); err != nil {
						return fmt.Errorf("set %s#%d: %w", e.Slug, setNum, err)
					}
				} else {
					reps := s.Reps
					if reps < 1 {
						reps = 1
					}
					if _, err := db.Exec(`INSERT INTO sets (exercise_id, name, set_number, rest_time, rep_count) VALUES ($1,$2,$3,$4,$5)`,
						exID, name, setNum, restNs, reps); err != nil {
						return fmt.Errorf("set %s#%d: %w", e.Slug, setNum, err)
					}
				}
			}
		}
		count++
	}

	fmt.Printf("exercises: %d categories, %d exercises seeded from %s\n", len(cats), count, base)
	return nil
}

func defaultSport(s string) string {
	if s == "" {
		return "CLIMBING"
	}
	return s
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// putMedia stores a seed file under exercises/<name> and returns its URL —
// through the same service the upload endpoint uses, so seeded and uploaded
// media are addressed identically wherever storage happens to live.
func putMedia(src, name string) (string, error) {
	ctx := context.Background()

	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return "", err
	}

	key := storage.KeyNamed(storage.KindExercise, name)
	if err := storage.Get().Put(ctx, key, in, info.Size(), mime.TypeByExtension(filepath.Ext(name))); err != nil {
		return "", err
	}
	return storage.Get().URL(key), nil
}
