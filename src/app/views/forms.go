package views

import (
	"coachwise/src/app/models"
	"time"

	"github.com/google/uuid"
)

type ExerciseForm struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	SportType   *models.ExerciseSportType `json:"sport_type"`
	MediaID     *uuid.UUID                `json:"media_id"`
	// Which extra metrics the exercise tracks. Pointers so an omitted flag falls
	// back to a default (weight on, the rest off) rather than forcing false.
	TrackWeight   *bool `json:"track_weight"`
	TrackDistance *bool `json:"track_distance"`
	TrackGrade    *bool `json:"track_grade"`
	TrackHeight   *bool `json:"track_height"`
	Sets          []struct {
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
	Distance        *float64   `json:"distance"`
	Height          *float64   `json:"height"`
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
	Distance        *float64 `json:"distance"`
	Height          *float64 `json:"height"`
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
	// Not required: a 0 (or omitted) rest is valid and clamped to a 1s minimum
	// by the handler, so `binding:"required"` must not reject it.
	RestTime  time.Duration `json:"rest_time"`
	Intensity int           `json:"intensity" binding:"required"`
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
	Name    string              `json:"name" binding:"required"`
	Records []models.SelfRecord `json:"records" binding:"required,min=1"`
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
	// Status must be one of the plan_schedule_status enum values; anything else
	// is rejected here rather than blowing up in Postgres.
	Status *string `json:"status" binding:"omitempty,oneof=ACTIVE CANCELED"`
	Notes  *string `json:"notes"`
}

// ProfileForm is a partial update: every field is optional, and only the fields
// actually present in the request body are applied. This prevents an omitted
// field (e.g. phone) from being blanked over the stored value. Client-side flows
// (onboarding) enforce their own required fields.
type ProfileForm struct {
	FirstName *string    `json:"first_name" binding:"omitempty,min=1"`
	LastName  *string    `json:"last_name" binding:"omitempty,min=1"`
	Username  *string    `json:"username" binding:"omitempty,min=5,max=24,alphanum"`
	JobTitle  *string    `json:"job_title"`
	Bio       *string    `json:"bio"`
	Phone     *string    `json:"phone"`
	AvatarID  *uuid.UUID `json:"avatar_id"`
	Website   *string    `json:"website"`
	Instagram *string    `json:"instagram"`
	// Birthday is an ISO date "YYYY-MM-DD" (or "" to clear).
	Birthday *string `json:"birthday"`
}

// Billing / wallet request bodies.
type DurationForm struct {
	Months int `json:"months" binding:"required,min=1"`
}

// PurchaseForm is the buy body: the chosen currency + provider, plus months for
// subscription purchases (ignored for one-time packages).
type PurchaseForm struct {
	Currency string `json:"currency" binding:"required"`
	Provider string `json:"provider" binding:"required"`
	Months   int    `json:"months"`
}

// PackagePriceForm sets a package's price in one currency (coach package builder).
type PackagePriceForm struct {
	Currency string `json:"currency" binding:"required"`
	Amount   int64  `json:"amount" binding:"required,min=1"`
}

type TopUpForm struct {
	Amount   int64  `json:"amount" binding:"required,min=1"`
	Currency string `json:"currency"`
	Provider string `json:"provider" binding:"required"`
}

type PayoutForm struct {
	Amount int64   `json:"amount" binding:"required,min=1"`
	Note   *string `json:"note"`
}

// TopUpInitiateForm starts a redirect (gateway) wallet top-up.
type TopUpInitiateForm struct {
	Amount   int64  `json:"amount" binding:"required,min=1"`
	Provider string `json:"provider" binding:"required"`
}

// PayoutAccountForm is the shared create/update body for a coach's payout
// destination. Which fields matter depends on the wallet currency: IRR uses
// CardNumber (+ optional AccountHolder); other currencies use the bank/Stripe
// fields once those methods are wired.
type PayoutAccountForm struct {
	AccountHolder *string `json:"account_holder"`
	CardNumber    *string `json:"card_number"`
	IBAN          *string `json:"iban"`
	BankName      *string `json:"bank_name"`
	Swift         *string `json:"swift"`
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

// OpenTicketForm opens a support ticket with its first message.
type OpenTicketForm struct {
	Subject string `json:"subject" binding:"required,max=140"`
	Body    string `json:"body" binding:"required,max=4000"`
}

// TicketMessageForm is a reply the user adds to an existing ticket.
type TicketMessageForm struct {
	Body string `json:"body" binding:"required,max=4000"`
}

// ClientErrorForm is a crash reported by the app (see telemetry.go). Everything
// is optional except the message: a report that arrives half-filled is still
// worth more than one that was rejected for being incomplete.
type ClientErrorForm struct {
	Message   string `json:"message" binding:"required,max=1000"`
	Stack     string `json:"stack" binding:"max=4000"`
	Kind      string `json:"kind" binding:"max=40"`     // crash | unhandled | api
	View      string `json:"view" binding:"max=80"`     // which screen it happened on
	URL       string `json:"url" binding:"max=300"`
	Version   string `json:"version" binding:"max=40"`
	Platform  string `json:"platform" binding:"max=40"` // web | android | ios
	Language  string `json:"language" binding:"max=10"`
	RequestID string `json:"request_id" binding:"max=64"` // the API request that failed, if any
}
