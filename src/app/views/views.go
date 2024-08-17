package views

import "github.com/gin-gonic/gin"

func Init(r *gin.Engine) {
	authGroup(r)
	userGroup(r)
	rootGroup(r)
}
