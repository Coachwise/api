package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/utils"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func authGroup(router *gin.Engine) {
	g := router.Group("auth")

	g.POST("/login", func(c *gin.Context) {
		form := new(auth.LoginForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "email or username required"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email/password not match"})
			return
		}
		if err := auth.CheckPasswordHash(form.Password, *u.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email/password not match"})
			return
		}

		tokens, err := auth.GenerateFullTokens(u.ID.String())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tokens["token"] = tokens["access_token"]
		c.JSON(http.StatusOK, tokens)
	})

	g.POST("/register", func(c *gin.Context) {
		form := new(auth.RegisterForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
			return
		}
		if _, err := models.GetUserByUsername(u.Username); err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
			return
		}

		ctx := c.MustGet("ctx")
		if err := u.Create(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := models.NewOTP(ctx.(context.Context), u.ID, "AUTH")

		if err != nil {
			fmt.Println("otp error:", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "Couldn't save OTP",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success", "id": u.ID})
	})

	g.POST("/refresh", func(c *gin.Context) {
		form := new(auth.RefreshTokenForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(form.RefreshToken) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}

		claims, err := auth.VerifyToken(form.RefreshToken)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		tb := models.TokenBlacklist{
			Token: form.RefreshToken,
		}
		ctx := c.MustGet("ctx")
		if err := tb.Create(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tokens, err := auth.GenerateFullTokens(claims.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.POST("/otp", func(c *gin.Context) {
		form := new(auth.OTPSendForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "User does not found",
			})
			return
		}

		ctx := c.MustGet("ctx")

		if _, err = models.NewOTP(ctx.(context.Context), u.ID, "AUTH"); err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "Couldn't save OTP",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.POST("/otp/verify", func(c *gin.Context) {
		form := new(auth.OTPConfirmForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		codeStr := ""
		switch v := form.Code.(type) {
		case string:
			codeStr = v
		case float64:
			codeStr = strconv.Itoa(int(v))
		case int:
			codeStr = strconv.Itoa(v)
		default:
			codeStr = fmt.Sprint(v)
		}
		if strings.TrimSpace(codeStr) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
			return
		}

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email/password not match"})
			return
		}

		otp, err := models.GetOTPByEmail(form.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
			return
		}

		if strconv.Itoa(otp.Code) != codeStr {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
			return
		}

		//For testing, if no OTP found just issue tokens
		ctx := c.MustGet("ctx")
		u.Status = "ACTIVE"
		_ = u.Verify(ctx.(context.Context))

		tokens, err := auth.GenerateFullTokens(u.ID.String())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, tokens)
	})

	g.POST("/password/forget", func(c *gin.Context) {

		form := new(auth.OTPSendForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "User does not found",
			})
			return
		}

		//Creating OTP
		ctx := c.MustGet("ctx")
		if _, err = models.NewOTP(ctx.(context.Context), u.ID, "FORGET_PASSWORD"); err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "Couldn't save OTP",
			})
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

			//Direct Password change
			form := new(auth.DirectPasswordChangeForm)
			if err := c.ShouldBindJSON(form); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			password = form.Password

		} else {

			//Normal Password change
			form := new(auth.NormalPasswordChangeForm)
			if err := c.ShouldBindJSON(form); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := auth.CheckPasswordHash(form.CurrentPassword, *user.Password); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "email/password not match"})
				return
			}
			password = form.Password
		}

		newPassword, err := auth.HashPassword(password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		user.Password = &newPassword
		if err := user.UpdatePassword(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"message": "success"})

	})

	g.POST("/pre-register", func(c *gin.Context) {

		form := new(auth.PreRegisterForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
