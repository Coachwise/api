package tests_test

import (
	"bytes"
	"coachwise/src/app/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func authGroup() {

	authExecuted = true

	It("should return status 200 with jwt tokens", func() {
		w := httptest.NewRecorder()
		reqBody, _ := json.Marshal(usersData[0])
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		body := decodeBody(w.Body)
		Expect(w.Code).To(Equal(200))
		bodyExpect(body, gin.H{"message": "success"})
	})

	It("Should return status 200 with jwt tokens", func() {
		//Get OTP
		otp := new(models.OTP)
		db.Get(otp, "SELECT * FROM otps LIMIT 1")
		w := httptest.NewRecorder()
		reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"], "code": otp.Code})
		req, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		body := decodeBody(w.Body)
		Expect(w.Code).To(Equal(200))
		bodyExpect(body, gin.H{"access_token": "<ANY>", "refresh_token": "<ANY>", "token_type": "Bearer"})
		authTokens = append(authTokens, body["access_token"].(string))
		authRefreshTokens = append(authRefreshTokens, body["refresh_token"].(string))
	})

	It("Should return status 200 with email and username avalibility status as existed", func() {
		w := httptest.NewRecorder()
		reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"], "username": usersData[0]["username"]})
		req, _ := http.NewRequest("POST", "/auth/pre-register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		body := decodeBody(w.Body)
		Expect(w.Code).To(Equal(200))
		bodyExpect(body, gin.H{"email": "EXISTS", "username": "EXISTS"})
	})

	It("Should return status 200 and create otp tokens", func() {
		w := httptest.NewRecorder()
		reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"]})
		req, _ := http.NewRequest("POST", "/auth/password/forget", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		body := decodeBody(w.Body)
		Expect(w.Code).To(Equal(200))
		bodyExpect(body, gin.H{"message": "success"})
	})

	It("Should return status 200 and update password", func() {
		w := httptest.NewRecorder()
		newPassword := "test1234567"
		reqBody, _ := json.Marshal(gin.H{"current_password": usersData[0]["password"], "password": newPassword})
		req, _ := http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(202))
	})

	It("Should return status 200 and generate jwt tokens", func() {
		w := httptest.NewRecorder()
		reqBody, _ := json.Marshal(gin.H{"refresh_token": authRefreshTokens[0]})
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		body := decodeBody(w.Body)
		Expect(w.Code).To(Equal(200))
		bodyExpect(body, gin.H{"access_token": "<ANY>", "refresh_token": "<ANY>", "token_type": "Bearer"})
	})

}
