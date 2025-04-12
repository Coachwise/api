package auth

import (
	"math"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type RegisterForm struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Username  *string `json:"username"`
	Email     string  `json:"email" validate:"required,email"`
	Password  *string `json:"password"`
}

type LoginForm struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type OTPSendForm struct {
	Email string `json:"email" validate:"required,email"`
}
type OTPConfirmForm struct {
	Email string `json:"email" validate:"required,email"`
	Code  int    `json:"code" validate:"required"`
}

type RefreshTokenForm struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type PreRegisterForm struct {
	Email    *string `json:"email" validate:"email"`
	Username *string `json:"username"`
}

type NormalPasswordChangeForm struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	Password        string `json:"password" validate:"required"`
}

type DirectPasswordChangeForm struct {
	Password string `json:"password" validate:"required"`
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func GenerateUsername(email string) string {
	var username string = email
	var re *regexp.Regexp

	re = regexp.MustCompile("@.*$")
	username = re.ReplaceAllString(username, "")

	re = regexp.MustCompile("[^a-z0-9._-]")
	username = re.ReplaceAllString(username, "-")

	re = regexp.MustCompile("[._-]{2,}")
	username = re.ReplaceAllString(username, "-")

	username = strings.ToLower(username)
	username = username[0:int(math.Min(float64(len(username)), 20))]

	username = username + strconv.Itoa(int(1000+rand.Float64()*9000))

	return username
}
