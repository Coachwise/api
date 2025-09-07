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

	Describe("Registration", func() {
		It("should register a new user successfully", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(usersData[0])
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"message": "success"})
		})

		It("should fail registration with duplicate email", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "Another",
				"last_name":  "User",
				"username":   "another",
				"email":      usersData[0]["email"], // duplicate email
				"password":   "password123",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail registration with invalid email format", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "Invalid",
				"last_name":  "Email",
				"username":   "invalidemail",
				"email":      "notanemail",
				"password":   "password123",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail registration with missing required fields", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"username": "incomplete",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("OTP Verification", func() {
		It("should verify OTP and return JWT tokens", func() {
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

		It("should fail OTP verification with invalid code", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"], "code": "000000"})
			req, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail OTP verification with non-existent email", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": "nonexistent@test.com", "code": "123456"})
			req, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Pre-Registration Check", func() {
		It("should check existing email and username", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"], "username": usersData[0]["username"]})
			req, _ := http.NewRequest("POST", "/auth/pre-register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"email": "EXISTS", "username": "EXISTS"})
		})

		It("should check available email and username", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": "new@test.com", "username": "newuser"})
			req, _ := http.NewRequest("POST", "/auth/pre-register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"email": "AVAILABLE", "username": "AVAILABLE"})
		})
	})

	Describe("Password Reset", func() {
		It("should initiate password reset for existing user", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"]})
			req, _ := http.NewRequest("POST", "/auth/password/forget", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"message": "success"})
		})

		It("should handle password reset for non-existent user", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": "nonexistent@test.com"})
			req, _ := http.NewRequest("POST", "/auth/password/forget", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			// Usually returns success to prevent email enumeration
			Expect(w.Code).To(BeNumerically(">=", 200))
			Expect(w.Code).To(BeNumerically("<", 500))
		})
	})

	Describe("Password Update", func() {
		It("should update password with valid current password", func() {
			w := httptest.NewRecorder()
			newPassword := "test1234567"
			reqBody, _ := json.Marshal(gin.H{"current_password": usersData[0]["password"], "password": newPassword})
			req, _ := http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(202))
			// Update the password in test data for future login tests
			usersData[0]["password"] = newPassword
		})

		It("should fail password update with incorrect current password", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"current_password": "wrongpassword", "password": "newpassword123"})
			req, _ := http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail password update without authentication", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"current_password": "anypassword", "password": "newpassword123"})
			req, _ := http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})

	Describe("Token Refresh", func() {
		It("should refresh tokens with valid refresh token", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"refresh_token": authRefreshTokens[0]})
			req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"access_token": "<ANY>", "refresh_token": "<ANY>", "token_type": "Bearer"})
		})

		It("should fail token refresh with invalid refresh token", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"refresh_token": "invalid.refresh.token"})
			req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("should fail token refresh with missing refresh token", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{})
			req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Login", func() {
		It("should login with valid credentials", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"email":    usersData[0]["email"],
				"password": usersData[0]["password"],
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"access_token": "<ANY>", "refresh_token": "<ANY>", "token_type": "Bearer"})
		})

		It("should fail login with invalid password", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"email":    usersData[0]["email"],
				"password": "wrongpassword",
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail login with non-existent email", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"email":    "nonexistent@test.com",
				"password": "anypassword",
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail login with missing credentials", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"email": usersData[0]["email"]})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Logout", func() {
		It("should logout successfully with valid token", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/auth/logout", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
		})

		It("should fail logout without authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/auth/logout", nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})

}
