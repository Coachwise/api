// Dev data seeder. Populates the database configured in config.yml with
// sample data for manual testing. It is built as a registry of named, idempotent
// seeders so new data sets (plans, exercises, ...) can be added without touching
// the runner.
//
// Run from api/:
//
//	go run cmd/seed/main.go                     # run every seeder
//	go run cmd/seed/main.go users               # run only the named seeder(s)
//	go run cmd/seed/main.go -dir /path exercises # override the seed-data dir
//
// Content-heavy seeders (e.g. exercises) read their data from files under the
// seed-data directory (default ../seed-data, i.e. the Coachwise workspace root)
// rather than being compiled in. See seedDataDir.
package main

import (
	"coachwise/src/config"
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// seedDataDir is the root of the file-based seed data (categories, exercises,
// media). Set via -dir; defaults to the workspace root next to api/.
var seedDataDir string

// Seeder is one named, idempotent unit of sample data. Add new ones to the
// registry below; they run in declaration order so later seeders may rely on
// earlier ones (e.g. connections rely on users existing).
type Seeder struct {
	Name string
	Run  func(db *sql.DB) error
}

var registry = []Seeder{
	{"users", seedUsers},
	{"exercises", seedExercises},
}

func main() {
	dir := flag.String("dir", "../seed-data", "path to the file-based seed-data directory")
	flag.Parse()
	seedDataDir = *dir

	config.Init("config.yml")

	db, err := sql.Open("postgres", config.Config.Database.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("cannot reach database: %v", err)
	}

	want := flag.Args() // optional subset of seeder names
	for _, s := range registry {
		if len(want) > 0 && !contains(want, s.Name) {
			continue
		}
		if err := s.Run(db); err != nil {
			log.Fatalf("seeder %q failed: %v", s.Name, err)
		}
	}
	fmt.Println("seeding done.")
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func userID(db *sql.DB, email string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	return id, err
}
