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
	FirstName *string `json:"first_name" binding:"omitempty"`
	LastName  *string `json:"last_name" binding:"omitempty"`
	FullName  *string `json:"full_name" binding:"required_without=FirstName"`
	Username  *string `json:"username"`
	Email     string  `json:"email" binding:"required,email"`
	Password  *string `json:"password" binding:"required,min=8"`
}

type LoginForm struct {
	Email    string `json:"email" binding:"required_without=Username,omitempty,email"`
	Username string `json:"username" binding:"required_without=Email"`
	Password string `json:"password" binding:"required,min=8"`
}

type OTPSendForm struct {
	Email string `json:"email" binding:"required,email"`
}
type OTPConfirmForm struct {
	Email string      `json:"email" binding:"required,email"`
	Code  interface{} `json:"code" binding:"required"`
}

type RefreshTokenForm struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// PhoneOTPForm requests an OTP for a phone (passwordless login/signup).
type PhoneOTPForm struct {
	Phone string `json:"phone" binding:"required,min=6"`
}

// PhoneVerifyForm confirms a phone OTP and issues tokens.
type PhoneVerifyForm struct {
	Phone string      `json:"phone" binding:"required,min=6"`
	Code  interface{} `json:"code" binding:"required"`
}

type PreRegisterForm struct {
	Email    *string `json:"email" binding:"omitempty,email"`
	Username *string `json:"username"`
}

type NormalPasswordChangeForm struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	Password        string `json:"password" binding:"required,min=8"`
}

type DirectPasswordChangeForm struct {
	Password string `json:"password" binding:"required,min=8"`
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
