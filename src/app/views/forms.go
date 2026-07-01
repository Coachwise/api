package views

import (
	"coachwise/src/app/models"
	"time"

	"github.com/google/uuid"
)

type ExerciseForm struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Public      bool                      `json:"public"`
	SportType   *models.ExerciseSportType `json:"sport_type"`
	MediaID     *uuid.UUID                `json:"media_id"`
	Sets        []struct {
		Name     string         `json:"name"`
		RestTime time.Duration  `json:"rest_time"`
		RepCount *int           `json:"rep_count"`
		Duration *time.Duration `json:"duration"`
	} `json:"sets"`
}

type CreateSessionForm struct {
	SessionType string     `json:"session_type" binding:"required"`
	PlanID      *uuid.UUID `json:"plan_id"`
	Notes       *string    `json:"notes"`
}

type UpdateSessionForm struct {
	Status    *string `json:"status"`
	Notes     *string `json:"notes"`
	Intensity int     `json:"intensity" binding:"required,min=1,max=10"`
	Quality   int     `json:"quality" binding:"required,min=1,max=5"`
}

type CreateWorkoutLogForm struct {
	SessionID       uuid.UUID  `json:"session_id" binding:"required"`
	ExerciseID      *uuid.UUID `json:"exercise_id"`
	ExerciseName    *string    `json:"exercise_name"`
	SetNumber       int        `json:"set_number" binding:"required"`
	Reps            *int       `json:"reps"`
	Weight          *float64   `json:"weight"`
	RPE             *float64   `json:"rpe"`
	DurationSeconds *int       `json:"duration_seconds"`
	Grade           *string    `json:"grade"`
	Completed       *bool      `json:"completed"`
	Attempts        *int       `json:"attempts"`
	Notes           *string    `json:"notes"`
	Tags            []string   `json:"tags"`
}

type UpdateWorkoutLogForm struct {
	Reps            *int     `json:"reps"`
	Weight          *float64 `json:"weight"`
	RPE             *float64 `json:"rpe"`
	DurationSeconds *int     `json:"duration_seconds"`
	Grade           *string  `json:"grade"`
	Completed       *bool    `json:"completed"`
	Attempts        *int     `json:"attempts"`
	Notes           *string  `json:"notes"`
}

type FeedMediaForm struct {
	Kind         string  `json:"kind" binding:"required,oneof=IMAGE VIDEO"`
	URL          string  `json:"url" binding:"required"`
	ThumbnailURL *string `json:"thumbnail_url"`
	OrderIndex   int     `json:"order_index"`
}

type FeedCreateForm struct {
	Body       *string         `json:"body" binding:"required_without=Media"`
	Location   *string         `json:"location"`
	Visibility *string         `json:"visibility" binding:"omitempty,oneof=PUBLIC PRIVATE"`
	Media      []FeedMediaForm `json:"media" binding:"dive"`
	Tags       []string        `json:"tags"`
}

type FeedCommentForm struct {
	Body string `json:"body" binding:"required"`
}

type MessageForm struct {
	RecipientID *string `json:"recipient_id"` // required for DIRECT send
	ChatID      *string `json:"chat_id"`      // required for CHANNEL send
	Body        string  `json:"body" binding:"required"`
	MediaID     *string `json:"media_id"`
}

type CreatePlanForm struct {
	Name   string `json:"name" binding:"required"`
	Public bool   `json:"public"`
}

type UpdatePlanForm struct {
	Name   *string `json:"name"`
	Public *bool   `json:"public"`
}

type PlanExerciseForm struct {
	ExerciseID    uuid.UUID     `json:"exercise_id" binding:"required"`
	ExerciseOrder int           `json:"exercise_order" binding:"required"`
	RestTime      time.Duration `json:"rest_time" binding:"required"`
	Intensity     int           `json:"intensity" binding:"required"`
}

type PlanAssignForm struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// PackageForm is shared by package create and update (the payloads match).
type PackageForm struct {
	Name             string      `json:"name" binding:"required"`
	Description      *string     `json:"description"`
	PriceMonthly     *int        `json:"price_monthly"`
	PriceAnnual      *int        `json:"price_annual"`
	PriceOneTime     *int        `json:"price_one_time"`
	TrialDays        int         `json:"trial_days"`
	CheckInFrequency *string     `json:"check_in_frequency"`
	VideoAccess      bool        `json:"video_access"`
	NutritionGuides  bool        `json:"nutrition_guides"`
	CustomFeatures   []string    `json:"custom_features"`
	IsActive         *bool       `json:"is_active"`
	Popular          bool        `json:"popular"`
	PlanIDs          []uuid.UUID `json:"plan_ids"`
}

type SetPackagePlansForm struct {
	PlanIDs []uuid.UUID `json:"plan_ids"`
}

// TestForm is shared by assessment-test create and update.
type TestForm struct {
	Name        string                 `json:"name" binding:"required"`
	Description *string                `json:"description"`
	Public      bool                   `json:"public"`
	Items       []models.TestItemInput `json:"items"`
}

type TestRequestForm struct {
	AthleteID uuid.UUID `json:"athlete_id" binding:"required"`
	Note      *string   `json:"note"`
}

type TestSubmitForm struct {
	Records []models.SubmittedRecord `json:"records"`
}

// SelfAssessmentForm is an athlete recording their own assessment.
type SelfAssessmentForm struct {
	Name    string                `json:"name" binding:"required"`
	Records []models.SelfRecord   `json:"records" binding:"required,min=1"`
}

type AchievementForm struct {
	AthleteID   uuid.UUID `json:"athlete_id" binding:"required"`
	Title       string    `json:"title" binding:"required"`
	Description *string   `json:"description"`
}

// AchievementLayoutForm curates a user's profile trophy case: an ordered list of
// item keys ("badge:<id>" / "record:<exercise_id>") plus the hidden ones.
type AchievementLayoutForm struct {
	Order  []string `json:"order"`
	Hidden []string `json:"hidden"`
}

type PlanScheduleCreateForm struct {
	PlanID       uuid.UUID `json:"plan_id" binding:"required"`
	ScheduledFor string    `json:"scheduled_for" binding:"required"` // YYYY-MM-DD
	Status       *string   `json:"status"`
	Notes        *string   `json:"notes"`
}

type PlanScheduleUpdateForm struct {
	Status *string `json:"status"`
	Notes  *string `json:"notes"`
}

type ProfileForm struct {
	FirstName string     `json:"first_name" binding:"required"`
	LastName  string     `json:"last_name" binding:"required"`
	JobTitle  string     `json:"job_title"`
	Bio       string     `json:"bio"`
	Phone     string     `json:"phone"`
	AvatarID  *uuid.UUID `json:"avatar_id"`
}

type CoachApplicationForm struct {
	FullName        string  `json:"full_name" binding:"required"`
	Specialty       string  `json:"specialty" binding:"required"`
	ExperienceYears int     `json:"experience_years"`
	Certifications  string  `json:"certifications" binding:"required"`
	Bio             *string `json:"bio"`
	Website         *string `json:"website"`
	Instagram       *string `json:"instagram"`
}
