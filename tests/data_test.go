package tests_test

import "github.com/gin-gonic/gin"

var users_data = []gin.H{
	{
		"first_name": "TestName",
		"last_name":  "TestLastName",
		"username":   "test",
		"email":      "test@test.com",
		"password":   "test123456",
	},
}
