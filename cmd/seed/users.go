package main

import (
	"database/sql"
	"fmt"
)

// 30 sample users. The first 5 are coaches.
var seedFirstNames = []string{
	"Sara", "Ali", "Mina", "Nima", "Lena", "Omid", "Reza", "Niloofar", "Kaveh", "Shirin",
	"Babak", "Yasmin", "Arash", "Parisa", "Hooman", "Maryam", "Sina", "Donya", "Farhad", "Roya",
	"Pouya", "Negar", "Saeed", "Tara", "Mehdi", "Golnaz", "Kian", "Setareh", "Amir", "Bahar",
}
var seedLastNames = []string{
	"Karimi", "Rezaei", "Tehrani", "Sadeghi", "Hosseini", "Bakhtiari", "Ahmadi", "Mohammadi", "Jafari", "Kazemi",
	"Moradi", "Akbari", "Ghorbani", "Najafi", "Rahimi", "Sharifi", "Eskandari", "Yazdani", "Norouzi", "Salehi",
	"Mousavi", "Ranjbar", "Zare", "Farahani", "Soltani", "Davari", "Mirzaei", "Hashemi", "Aghaei", "Bahrami",
}

const seedCoachCount = 5

// sports enum values cycled across the coaches.
var seedSpecialties = []string{"FITNESS", "CLIMBING", "THERAPEUTIC", "FITNESS", "CLIMBING"}

// seedUserEmail is the deterministic email for the i-th sample user, so other
// seeders can reference users without re-deriving the scheme.
func seedUserEmail(i int) string {
	return fmt.Sprintf("%s@coachwise.test", seedUsername(i))
}

func seedUsername(i int) string {
	return lowerASCII(fmt.Sprintf("%s_%s", seedFirstNames[i], seedLastNames[i]))
}

func seedUsers(db *sql.DB) error {
	// Reuse an existing password hash so sample users share a known password
	// and can log in if needed. Falls back to NULL when no donor exists.
	var pwHash sql.NullString
	_ = db.QueryRow(`SELECT password FROM users WHERE password IS NOT NULL LIMIT 1`).Scan(&pwHash)

	coaches := 0
	for i := 0; i < len(seedFirstNames); i++ {
		var id string
		err := db.QueryRow(`
			INSERT INTO users (username, first_name, last_name, email, password, status)
			VALUES ($1, $2, $3, $4, $5, 'ACTIVE')
			ON CONFLICT (email) DO UPDATE SET first_name = EXCLUDED.first_name
			RETURNING id`,
			seedUsername(i), seedFirstNames[i], seedLastNames[i], seedUserEmail(i), pwHash).Scan(&id)
		if err != nil {
			return fmt.Errorf("user %s: %w", seedUsername(i), err)
		}

		if i < seedCoachCount {
			if _, err := db.Exec(`
				INSERT INTO coaches (user_id, specialties)
				VALUES ($1, ARRAY[$2]::sports[])
				ON CONFLICT DO NOTHING`, id, seedSpecialties[i]); err != nil {
				return fmt.Errorf("coach %s: %w", seedUsername(i), err)
			}
			coaches++
		}
	}

	fmt.Printf("users: %d seeded (%d coaches)\n", len(seedFirstNames), coaches)
	return nil
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
