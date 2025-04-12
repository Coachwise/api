package tests_test

import "github.com/gin-gonic/gin"

var (
	authTokens        = []string{}
	authRefreshTokens = []string{}

	usersData = []gin.H{
		{
			"first_name": "TestName",
			"last_name":  "TestLastName",
			"username":   "test",
			"email":      "test@test.com",
			"password":   "test123456",
		},
	}

	exercisesData = []gin.H{
		{
			"name":        "test",
			"description": "test",
			"public":      true,
			"sets": []gin.H{
				{"name": "test", "rest_time": 30e9, "rep_count": 6},
				{"name": "test", "rest_time": 40e9, "duration": 3e9},
			},
		},
	}
)
