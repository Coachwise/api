package main

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// The showcase story, for screenshots and for demoing the app: one coach, one
// athlete, and enough real history between them that every screen has something
// to say — plans assigned, sessions logged, an assessment protocol run three
// times with a personal record to show for it.
//
// It is deliberately its OWN pair of accounts rather than anyone's real one: the
// screenshots on the marketing site come from here, and nobody's name, phone or
// email belongs on a public page.
//
// Idempotent: everything is keyed and upserted, so re-running refreshes the story
// instead of duplicating it.
const (
	demoCoachEmail   = "demo.coach@coachwise.test"
	demoAthleteEmail = "demo.athlete@coachwise.test"
	demoPassword     = "Demo123456!"
)

func seedDemo(db *sql.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	coachID, err := demoUser(db, string(hash), demoCoachEmail, "democoach", "سارا", "کریمی", true)
	if err != nil {
		return fmt.Errorf("demo coach: %w", err)
	}
	athleteID, err := demoUser(db, string(hash), demoAthleteEmail, "demoathlete", "امید", "بختیاری", false)
	if err != nil {
		return fmt.Errorf("demo athlete: %w", err)
	}

	// They know each other — most of the app is gated behind a connection.
	if _, err := db.Exec(`
		INSERT INTO connection_requests (requester_id, addressee_id, status)
		VALUES ($1, $2, 'ACCEPTED')
		ON CONFLICT DO NOTHING`, coachID, athleteID); err != nil {
		return fmt.Errorf("connection: %w", err)
	}

	// A few climbing exercises to hang the story on.
	exercises, err := demoExercises(db, 6)
	if err != nil {
		return err
	}
	if len(exercises) < 3 {
		return fmt.Errorf("demo: need at least 3 exercises seeded first (run the exercises seeder)")
	}

	planID, err := demoPlan(db, coachID, athleteID, "بلوک قدرت انگشت", exercises[:3])
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	if _, err := demoPlan(db, coachID, athleteID, "استقامت و ریکاوری", exercises[3:]); err != nil {
		return fmt.Errorf("plan 2: %w", err)
	}

	if err := demoSchedule(db, athleteID, planID); err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	if err := demoSessions(db, athleteID, planID, exercises); err != nil {
		return fmt.Errorf("sessions: %w", err)
	}
	if err := demoAssessment(db, coachID, athleteID, exercises); err != nil {
		return fmt.Errorf("assessment: %w", err)
	}
	if err := demoPackage(db, coachID, athleteID, planID); err != nil {
		return fmt.Errorf("package: %w", err)
	}
	return nil
}

// demoSchedule puts training on the calendar, including today — otherwise the
// athlete's home screen is an empty rest day.
func demoSchedule(db *sql.DB, athleteID, planID string) error {
	if _, err := db.Exec(`DELETE FROM plan_schedules WHERE user_id = $1`, athleteID); err != nil {
		return err
	}
	for _, offset := range []int{0, 2, 4, 7} {
		day := time.Now().AddDate(0, 0, offset)
		if _, err := db.Exec(`
			INSERT INTO plan_schedules (user_id, plan_id, scheduled_for, status)
			VALUES ($1, $2, $3, 'ACTIVE')`,
			athleteID, planID, day.Format("2006-01-02")); err != nil {
			return err
		}
	}
	return nil
}

// demoPackage makes the athlete a paying client, so the coach's dashboard has a
// client and a package to show rather than two zeroes.
func demoPackage(db *sql.DB, coachID, athleteID, planID string) error {
	const name = "همراهی سنگ‌نوردی — ماهانه"

	var pkgID string
	if err := db.QueryRow(`SELECT id FROM coach_packages WHERE coach_id = $1 AND name = $2 AND deleted_at IS NULL`,
		coachID, name).Scan(&pkgID); err == sql.ErrNoRows {
		if err := db.QueryRow(`
			INSERT INTO coach_packages (coach_id, name, description, price_monthly, currency, is_active, popular, check_in_frequency)
			VALUES ($1, $2, 'برنامه‌ی اختصاصی، بازبینی هفتگی ویدیو، و ارزیابی ماهانه‌ی قدرت انگشت.', 1200000, 'IRR', true, true, 'WEEKLY')
			RETURNING id`, coachID, name).Scan(&pkgID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO coach_package_plans (package_id, plan_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, pkgID, planID); err != nil {
		return err
	}
	_, err := db.Exec(`
		INSERT INTO package_subscriptions (package_id, coach_id, client_id, status, ends_at)
		VALUES ($1, $2, $3, 'ACTIVE', now() + interval '1 month')
		ON CONFLICT (package_id, client_id) DO UPDATE
		    SET status = 'ACTIVE', deleted_at = NULL, ends_at = EXCLUDED.ends_at`,
		pkgID, coachID, athleteID)
	return err
}

func demoUser(db *sql.DB, hash, email, username, first, last string, coach bool) (string, error) {
	var id string
	err := db.QueryRow(`
		INSERT INTO users (username, first_name, last_name, email, password, status, is_coach)
		VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6)
		ON CONFLICT (email) DO UPDATE
		    SET first_name = EXCLUDED.first_name,
		        last_name  = EXCLUDED.last_name,
		        password   = EXCLUDED.password,
		        is_coach   = EXCLUDED.is_coach,
		        deleted_at = NULL
		RETURNING id`, username, first, last, email, hash, coach).Scan(&id)
	return id, err
}

// demoExercises picks public climbing exercises from the seeded library.
func demoExercises(db *sql.DB, n int) ([]string, error) {
	rows, err := db.Query(`
		SELECT id FROM exercises
		WHERE public = true AND deleted_at IS NULL
		ORDER BY sport_type = 'CLIMBING' DESC, created_at
		LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func demoPlan(db *sql.DB, coachID, athleteID, name string, exercises []string) (string, error) {
	var planID string
	if err := db.QueryRow(`SELECT id FROM plans WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL`,
		coachID, name).Scan(&planID); err == sql.ErrNoRows {
		if err := db.QueryRow(`
			INSERT INTO plans (user_id, name, public) VALUES ($1, $2, false) RETURNING id`,
			coachID, name).Scan(&planID); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	if _, err := db.Exec(`DELETE FROM plan_exercises WHERE plan_id = $1`, planID); err != nil {
		return "", err
	}
	for i, ex := range exercises {
		if _, err := db.Exec(`
			INSERT INTO plan_exercises (plan_id, exercise_id, exercise_order, rest_time, intensity)
			VALUES ($1, $2, $3, $4, $5)`,
			planID, ex, i+1, int64(180)*1e9, 7+i%3); err != nil {
			return "", err
		}
	}

	// Assigned to the athlete by the coach — this is what puts it on their board.
	if _, err := db.Exec(`
		INSERT INTO plan_assignees (plan_id, user_id, assigner)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, planID, athleteID, coachID); err != nil {
		return "", err
	}
	return planID, nil
}

// demoSessions writes a few weeks of finished training, each with the sets that
// were actually logged — the history the progress screens are drawn from.
func demoSessions(db *sql.DB, athleteID, planID string, exercises []string) error {
	if _, err := db.Exec(`
		DELETE FROM workout_logs WHERE session_id IN (SELECT id FROM sessions WHERE user_id = $1 AND notes = 'demo')`,
		athleteID); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM sessions WHERE user_id = $1 AND notes = 'demo'`, athleteID); err != nil {
		return err
	}

	// Six sessions over three weeks, getting heavier — a visible upward line.
	for i := 0; i < 6; i++ {
		day := time.Now().AddDate(0, 0, -(18 - i*3))
		var sessionID string
		if err := db.QueryRow(`
			INSERT INTO sessions (user_id, session_type, plan_id, status, started_at, ended_at, notes, intensity, quality)
			VALUES ($1, 'CLIMBING', $2, 'COMPLETED', $3, $4, 'demo', $5, $6)
			RETURNING id`,
			athleteID, planID, day, day.Add(72*time.Minute), 6+i%4, 3+i%3).Scan(&sessionID); err != nil {
			return err
		}
		for e, ex := range exercises[:3] {
			for set := 1; set <= 3; set++ {
				weight := 20 + i*2 + set // creeping up week on week
				if _, err := db.Exec(`
					INSERT INTO workout_logs
					    (session_id, exercise_id, set_number, reps, weight, rpe, completed, logged_at)
					VALUES ($1, $2, $3, $4, $5, $6, true, $7)`,
					sessionID, ex, set, 8-set, weight, 7+float64(set)*0.5, day.Add(time.Duration(e*10+set)*time.Minute)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// demoAssessment builds the protocol story end to end: the coach writes a test,
// assigns it, and the athlete runs it three times — so the history has a shape
// and the last run is a personal record.
func demoAssessment(db *sql.DB, coachID, athleteID string, exercises []string) error {
	const name = "تست قدرت انگشت — آویزان ۲۰ میلی‌متر"

	var testID string
	if err := db.QueryRow(`SELECT id FROM tests WHERE coach_id = $1 AND name = $2 AND deleted_at IS NULL`,
		coachID, name).Scan(&testID); err == sql.ErrNoRows {
		if err := db.QueryRow(`
			INSERT INTO tests (coach_id, name, description, public)
			VALUES ($1, $2, 'سه تلاش بیشینه روی لبه‌ی ۲۰ میلی‌متری، با ۳ دقیقه استراحت.', false)
			RETURNING id`, coachID, name).Scan(&testID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := db.Exec(`DELETE FROM test_items WHERE test_id = $1`, testID); err != nil {
		return err
	}
	for i, ex := range exercises[:2] {
		if _, err := db.Exec(`
			INSERT INTO test_items (test_id, exercise_id, item_order, track_reps, track_weight, track_time)
			VALUES ($1, $2, $3, false, true, true)`, testID, ex, i+1); err != nil {
			return err
		}
	}

	// The athlete's own protocol, which they wrote and re-run on themselves —
	// assessments aren't only something a coach hands down.
	const own = "چک‌این هفتگی خودم"
	var ownID string
	if err := db.QueryRow(`SELECT id FROM tests WHERE coach_id = $1 AND name = $2 AND deleted_at IS NULL`,
		athleteID, own).Scan(&ownID); err == sql.ErrNoRows {
		if err := db.QueryRow(`
			INSERT INTO tests (coach_id, name, description, public)
			VALUES ($1, $2, 'هر شنبه: کشش بیشینه و آویزان تا واماندگی.', false)
			RETURNING id`, athleteID, own).Scan(&ownID); err != nil {
			return err
		}
		if _, err := db.Exec(`
			INSERT INTO test_items (test_id, exercise_id, item_order, track_reps, track_weight, track_time)
			VALUES ($1, $2, 1, true, true, false)`, ownID, exercises[2]); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Three dated runs, each better than the last.
	if _, err := db.Exec(`
		DELETE FROM workout_logs WHERE test_request_id IN
		    (SELECT id FROM test_requests WHERE athlete_id = $1 AND test_id = $2)`, athleteID, testID); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM test_requests WHERE athlete_id = $1 AND test_id = $2`, athleteID, testID); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		day := time.Now().AddDate(0, 0, -(35 - i*14))
		var reqID string
		if err := db.QueryRow(`
			INSERT INTO test_requests (test_id, coach_id, athlete_id, status, submitted_at, seen_at, created_at)
			VALUES ($1, $2, $3, 'SEEN', $4, $4, $4)
			RETURNING id`, testID, coachID, athleteID, day).Scan(&reqID); err != nil {
			return err
		}
		for e, ex := range exercises[:2] {
			if _, err := db.Exec(`
				INSERT INTO workout_logs
				    (test_request_id, exercise_id, set_number, weight, duration_seconds, completed, logged_at)
				VALUES ($1, $2, 1, $3, $4, true, $5)`,
				reqID, ex, 24+i*4+e*2, 10+i*2, day); err != nil {
				return err
			}
		}
	}
	return nil
}
