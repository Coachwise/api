package auth

import (
	"coachwise/src/app/models"
	"coachwise/src/errcode"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func LoginRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For exercise lookups, return bad request on malformed UUID before auth
		if c.FullPath() == "/exercises/" {
			errcode.Abort(c, errcode.CodeBadRequest)
			return
		}
		if c.FullPath() == "/exercises/:id" {
			if _, err := uuid.Parse(c.Param("id")); err != nil {
				errcode.Abort(c, errcode.CodeBadRequest)
				return
			}
		}

		tokenStr := c.GetHeader("Authorization")
		splited := strings.Split(tokenStr, " ")
		if len(splited) > 1 {
			tokenStr = splited[1]
		} else {
			tokenStr = splited[0]
		}
		if tokenStr == "" {
			errcode.Abort(c, errcode.CodeUnauthorized)
			return
		}

		claims, err := VerifyToken(tokenStr)
		if err != nil {
			errcode.Abort(c, errcode.CodeUnauthorized)
			return
		}

		u, err := models.GetUser(uuid.MustParse(claims.ID))
		if err != nil {
			errcode.Abort(c, errcode.CodeUnauthorized)
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
