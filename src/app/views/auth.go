package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"coachwise/src/utils"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// otpCodeInt normalizes a JSON code (string or number) to an int.
func otpCodeInt(v interface{}) (int, bool) {
	var s string
	switch t := v.(type) {
	case string:
		s = t
	case float64:
		s = strconv.Itoa(int(t))
	case int:
		s = strconv.Itoa(t)
	default:
		s = fmt.Sprint(t)
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, err == nil
}

func authGroup(router *gin.Engine) {
	g := router.Group("auth")

	g.POST("/login", func(c *gin.Context) {
		form := new(auth.LoginForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		var (
			u   *models.User
			err error
		)
		if form.Email != "" {
			u, err = models.GetUserByEmail(form.Email)
		} else if form.Username != "" {
			// Allow username field to carry either username or email
			u, err = models.GetUserByUsername(form.Username)
			if err != nil {
				u, err = models.GetUserByEmail(form.Username)
			}
		} else {
			Abort(c, CodeBadRequest)
			return
		}
		if err != nil {
			Abort(c, CodeInvalidCredentials)
			return
		}
		if err := auth.CheckPasswordHash(form.Password, *u.Password); err != nil {
			Abort(c, CodeInvalidCredentials)
			return
		}

		tokens, err := auth.GenerateFullTokens(u.ID.String())
		if err != nil {
			Abort(c, CodeTokenGeneration)
			return
		}

		tokens["token"] = tokens["access_token"]
		c.JSON(http.StatusOK, tokens)
	})

	g.POST("/register", func(c *gin.Context) {
		form := new(auth.RegisterForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		u := new(models.User)
		utils.Copy(form, u)
		// Split full_name into first/last if provided
		if form.FullName != nil && (form.FirstName == nil && form.LastName == nil) {
			parts := strings.Fields(*form.FullName)
			if len(parts) > 0 {
				first := parts[0]
				form.FirstName = &first
			}
			if len(parts) > 1 {
				last := strings.Join(parts[1:], " ")
				form.LastName = &last
			}
		}

		if form.Password != nil {
			password, _ := auth.HashPassword(*form.Password)
			u.Password = &password
		}

		if form.Username == nil {
			u.Username = auth.GenerateUsername(u.Email)
		} else {
			u.Username = *form.Username
		}

		// Avoid hitting unique constraints repeatedly so we don't trip circuit breakers
		if _, err := models.GetUserByEmail(u.Email); err == nil {
			Abort(c, CodeEmailExists)
			return
		}
		if _, err := models.GetUserByUsername(u.Username); err == nil {
			Abort(c, CodeUsernameExists)
			return
		}

		ctx := c.MustGet("ctx")
		if err := u.Create(ctx.(context.Context)); err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
			return
		}

		if _, err := models.NewOTP(ctx.(context.Context), u.ID, "AUTH"); err != nil {
			AbortMsg(c, CodeOTPSendFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success", "id": u.ID})
	})

	g.POST("/refresh", func(c *gin.Context) {
		form := new(auth.RefreshTokenForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if strings.TrimSpace(form.RefreshToken) == "" {
			Abort(c, CodeRefreshInvalid)
			return
		}

		claims, err := auth.VerifyToken(form.RefreshToken)
		if err != nil {
			Abort(c, CodeRefreshInvalid)
			return
		}
		// Only an actual refresh token can be exchanged — not an access token.
		if !claims.Refresh {
			Abort(c, CodeRefreshInvalid)
			return
		}

		tb := models.TokenBlacklist{Token: form.RefreshToken}
		ctx := c.MustGet("ctx")
		if err := tb.Create(ctx.(context.Context)); err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
			return
		}

		tokens, err := auth.GenerateFullTokens(claims.ID)
		if err != nil {
			Abort(c, CodeTokenGeneration)
			return
		}

		c.JSON(http.StatusOK, tokens)
	})

	g.POST("/logout", auth.LoginRequired(), func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := strings.Split(authHeader, " ")
		tokenStr := tokenParts[len(tokenParts)-1]
		ctx := c.MustGet("ctx")
		tb := models.TokenBlacklist{Token: tokenStr}
		if err := tb.Create(ctx.(context.Context)); err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.POST("/otp", rateLimit(5, time.Minute), func(c *gin.Context) {
		form := new(auth.OTPSendForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			Abort(c, CodeUserNotFound)
			return
		}

		ctx := c.MustGet("ctx")
		_, err = models.NewOTP(ctx.(context.Context), u.ID, "AUTH")
		if err == models.ErrOTPCooldown {
			Abort(c, CodeOTPCooldown)
			return
		}
		if err != nil {
			Abort(c, CodeOTPSendFailed)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.POST("/otp/verify", func(c *gin.Context) {
		form := new(auth.OTPConfirmForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		code, ok := otpCodeInt(form.Code)
		if !ok {
			Abort(c, CodeOTPInvalid)
			return
		}

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			Abort(c, CodeOTPInvalid)
			return
		}

		ctx := c.MustGet("ctx")
		// Validate + consume atomically (checks expiry, single-use).
		valid, err := models.ConsumeOTP(ctx.(context.Context), u.ID, "AUTH", code)
		if err != nil || !valid {
			Abort(c, CodeOTPInvalid)
			return
		}

		u.Status = "ACTIVE"
		_ = u.Verify(ctx.(context.Context))

		tokens, err := auth.GenerateFullTokens(u.ID.String())
		if err != nil {
			Abort(c, CodeTokenGeneration)
			return
		}

		c.JSON(http.StatusOK, tokens)
	})

	// Passwordless phone login: send an OTP (creating the account on first use).
	// Rate-limited per IP; the model also enforces a per-phone cooldown.
	g.POST("/phone/otp", rateLimit(5, time.Minute), func(c *gin.Context) {
		form := new(auth.PhoneOTPForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		u, err := models.GetOrCreatePhoneUser(ctx, strings.TrimSpace(form.Phone))
		if err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
			return
		}
		otp, err := models.NewOTP(ctx, u.ID, "AUTH")
		if err == models.ErrOTPCooldown {
			Abort(c, CodeOTPCooldown)
			return
		}
		if err != nil {
			Abort(c, CodeOTPSendFailed)
			return
		}
		// Queue the OTP so this request returns fast; a worker delivers it via the
		// country's SMS provider (Kavenegar verify template for Iran).
		events.EmitOTP(form.Phone, fmt.Sprintf("%d", otp.Code))
		c.JSON(http.StatusOK, gin.H{"message": "code sent"})
	})

	// Confirm a phone OTP and issue tokens.
	g.POST("/phone/verify", func(c *gin.Context) {
		form := new(auth.PhoneVerifyForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		code, ok := otpCodeInt(form.Code)
		if !ok {
			Abort(c, CodeOTPInvalid)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		u, err := models.GetUserByPhone(strings.TrimSpace(form.Phone))
		if err != nil {
			Abort(c, CodeOTPInvalid)
			return
		}
		valid, err := models.ConsumeOTP(ctx, u.ID, "AUTH", code)
		if err != nil || !valid {
			Abort(c, CodeOTPInvalid)
			return
		}
		u.Status = "ACTIVE"
		_ = u.Verify(ctx)
		tokens, err := auth.GenerateFullTokens(u.ID.String())
		if err != nil {
			Abort(c, CodeTokenGeneration)
			return
		}
		tokens["token"] = tokens["access_token"]
		c.JSON(http.StatusOK, tokens)
	})

	g.POST("/password/forget", rateLimit(5, time.Minute), func(c *gin.Context) {
		form := new(auth.OTPSendForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			Abort(c, CodeUserNotFound)
			return
		}

		// Creating OTP
		ctx := c.MustGet("ctx")
		_, err = models.NewOTP(ctx.(context.Context), u.ID, "FORGET_PASSWORD")
		if err == models.ErrOTPCooldown {
			Abort(c, CodeOTPCooldown)
			return
		}
		if err != nil {
			Abort(c, CodeOTPSendFailed)
			return
		}

		// Mark password as expired so PUT /auth/password accepts a new password without current_password
		_ = u.ExpirePassword(ctx.(context.Context))

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.PUT("/password", auth.LoginRequired(), func(c *gin.Context) {
		ctx := c.MustGet("ctx")
		user := c.MustGet("user").(*models.User)
		var password string

		if user.PasswordExpired || user.Password == nil {
			// Direct password change
			form := new(auth.DirectPasswordChangeForm)
			if err := c.ShouldBindJSON(form); err != nil {
				AbortValidation(c, err)
				return
			}
			password = form.Password
		} else {
			// Normal password change
			form := new(auth.NormalPasswordChangeForm)
			if err := c.ShouldBindJSON(form); err != nil {
				AbortValidation(c, err)
				return
			}
			if err := auth.CheckPasswordHash(form.CurrentPassword, *user.Password); err != nil {
				Abort(c, CodeInvalidCredentials)
				return
			}
			password = form.Password
		}

		newPassword, err := auth.HashPassword(password)
		if err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
			return
		}

		user.Password = &newPassword
		if err := user.UpdatePassword(ctx.(context.Context)); err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"message": "success"})
	})

	g.POST("/pre-register", func(c *gin.Context) {
		form := new(auth.PreRegisterForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		emailStatus := "UNKOWN"
		usernameStatus := "UNKOWN"

		if form.Email != nil {
			emailStatus = "AVAILABLE"
			if _, err := models.GetUserByEmail(*form.Email); err == nil {
				emailStatus = "EXISTS"
			}
		}
		if form.Username != nil {
			usernameStatus = "AVAILABLE"
			if _, err := models.GetUserByUsername(*form.Username); err == nil {
				usernameStatus = "EXISTS"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"email":    emailStatus,
			"username": usernameStatus,
		})
	})
}
