package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/utils"
	"context"
	"net/http"
	"time"

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
		u, err := models.GetUserByEmail(form.Email)
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
		if form.Password != nil {
			password, _ := auth.HashPassword(*form.Password)
			u.Password = &password
		}

		if form.Username == nil {
			u.Username = auth.GenerateUsername(u.Email)
		}

		ctx, _ := c.Get("ctx")
		if err := u.Create(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := models.NewOTP(ctx.(context.Context), u.ID, "AUTH")

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "Couldn't save OTP",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	g.POST("/refresh", func(c *gin.Context) {
		form := new(auth.RefreshTokenForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		claims, err := auth.VerifyToken(form.RefreshToken)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tb := models.TokenBlacklist{
			Token: form.RefreshToken,
		}
		ctx, _ := c.Get("ctx")
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

		ctx, _ := c.Get("ctx")

		otp, err := models.GetOTPByUserID(u.ID)
		if err == nil {
			if time.Now().Before(otp.ExpiresAt) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Code exists",
					"message": "Can't send code before expiration",
				})
				return
			}
		}

		otp, err = models.NewOTP(ctx.(context.Context), u.ID, "AUTH")
		if err != nil {
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

		u, err := models.GetUserByEmail(form.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email/password not match"})
			return
		}

		//Verifying OTP
		ctx, _ := c.Get("ctx")
		otp := models.OTP{
			UserID: u.ID,
			Code:   form.Code,
		}

		err = otp.Verify(ctx.(context.Context))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   err.Error(),
				"message": "A problem occured when trying to verify the code",
			})
			return
		}
		if !otp.IsVerified {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   nil,
				"message": "Code does not found or it is wrong",
			})
			return
		}

		//Verifying User
		u.Status = "ACTIVE"
		if err := u.Verify(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if otp.Perpose == "FORGET_PASSWORD" {
			if err := u.ExpirePassword(ctx.(context.Context)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		//Generating Token
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
		ctx, _ := c.Get("ctx")
		_, err = models.NewOTP(ctx.(context.Context), u.ID, "FORGET_PASSWORD")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   err.Error(),
				"message": "Couldn't save OTP",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})

	})

	g.PUT("/password", auth.LoginRequired(), func(c *gin.Context) {

		ctx, _ := c.Get("ctx")
		u, _ := c.Get("user")
		var password string
		user := u.(*models.User)
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
			if err := auth.CheckPasswordHash(form.CurrentPassword, *u.(*models.User).Password); err != nil {
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
		if err := u.(*models.User).UpdatePassword(ctx.(context.Context)); err != nil {
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
			u, err := models.GetUserByEmail(*form.Email)
			emailStatus = "AVAILABLE"
			if err == nil && u.Status == "ACTIVE" {
				emailStatus = "EXISTS"
			}
		}
		if form.Username != nil {
			u, err := models.GetUserByUsername(*form.Username)
			usernameStatus = "AVAILABLE"
			if err == nil && u.Status == "ACTIVE" {
				usernameStatus = "EXISTS"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"email":    emailStatus,
			"username": usernameStatus,
		})

	})

}
