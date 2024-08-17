package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func userGroup(router *gin.Engine) {
	g := router.Group("users")
	g.Use(auth.LoginRequired())

	g.GET("/", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		u, err := models.GetUser(uuid.MustParse(userID.(string)))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, u)
	})
}
