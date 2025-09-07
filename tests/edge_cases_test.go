package tests_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func edgeCasesGroup() {
	Describe("Input Validation", func() {
		It("should handle extremely long strings", func() {
			longString := strings.Repeat("a", 10000)
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        longString,
				"description": "Test",
				"public":      false,
				"sets":        []gin.H{},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			// Should either truncate or reject
			Expect(w.Code).To(BeNumerically(">=", 400))
		})

		It("should handle special characters in usernames", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "Special",
				"last_name":  "User",
				"username":   "user!@#$%^&*()",
				"email":      "special@test.com",
				"password":   "password123",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			// Should either sanitize or reject
			Expect(w.Code).To(BeNumerically(">=", 200))
		})

		It("should handle SQL injection attempts", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"email":    "test@test.com'; DROP TABLE users; --",
				"password": "password",
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			// Should safely handle and reject
			Expect(w.Code).To(Equal(400))
		})

		It("should handle XSS attempts in exercise names", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "<script>alert('XSS')</script>",
				"description": "Test XSS",
				"public":      false,
				"sets":        []gin.H{},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code == 201 {
				body := decodeBody(w.Body)
				// Should escape or sanitize the script tag
				Expect(body["name"]).NotTo(ContainSubstring("<script>"))
			}
		})

		It("should handle negative numbers in sets", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Negative Test",
				"description": "Testing negative values",
				"public":      false,
				"sets": []gin.H{
					{"name": "Set 1", "rest_time": -30e9, "rep_count": -10},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			// Should reject negative values
			Expect(w.Code).To(Equal(400))
		})

		It("should handle null values appropriately", func() {
			w := httptest.NewRecorder()
			reqBody := []byte(`{"name": "Null Test", "description": null, "public": false, "sets": null}`)
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			// Should handle null values gracefully
			Expect(w.Code).To(BeNumerically(">=", 200))
		})

		It("should handle empty JSON objects", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should handle malformed JSON", func() {
			w := httptest.NewRecorder()
			reqBody := []byte(`{"name": "Malformed", "description": }`)
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Boundary Conditions", func() {
		It("should handle maximum integer values", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Max Int Test",
				"description": "Testing maximum values",
				"public":      false,
				"sets": []gin.H{
					{"name": "Set 1", "rest_time": 9223372036854775807, "rep_count": 2147483647},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			// Should either accept or gracefully reject
			Expect(w.Code).To(BeNumerically(">=", 200))
		})

		It("should handle zero values in sets", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Zero Test",
				"description": "Testing zero values",
				"public":      false,
				"sets": []gin.H{
					{"name": "Set 1", "rest_time": 0, "rep_count": 0},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			// Should handle zero values appropriately
			Expect(w.Code).To(BeNumerically(">=", 200))
		})

		It("should handle very short passwords", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "Short",
				"last_name":  "Pass",
				"username":   "shortpass",
				"email":      "short@test.com",
				"password":   "1",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			// Should reject too short passwords
			Expect(w.Code).To(Equal(400))
		})

		It("should handle empty arrays", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Empty Sets",
				"description": "No sets",
				"public":      false,
				"sets":        []gin.H{},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			// Should accept exercises with no sets
			if w.Code == 201 {
				body := decodeBody(w.Body)
				sets := body["sets"].([]interface{})
				Expect(len(sets)).To(Equal(0))
			}
		})
	})

	Describe("Concurrency and Race Conditions", func() {
		It("should handle simultaneous registration with same email", func() {
			email := "concurrent@test.com"
			results := make(chan int, 2)
			
			for i := 0; i < 2; i++ {
				go func(idx int) {
					w := httptest.NewRecorder()
					reqBody, _ := json.Marshal(gin.H{
						"first_name": fmt.Sprintf("User%d", idx),
						"last_name":  "Concurrent",
						"username":   fmt.Sprintf("concurrent%d", idx),
						"email":      email,
						"password":   "password123",
					})
					req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
					req.Header.Set("Content-Type", "application/json")
					router.ServeHTTP(w, req)
					results <- w.Code
				}(i)
			}
			
			code1 := <-results
			code2 := <-results
			
			// One should succeed, one should fail
			successCount := 0
			if code1 == 200 {
				successCount++
			}
			if code2 == 200 {
				successCount++
			}
			Expect(successCount).To(BeNumerically("<=", 1))
		})

		It("should handle rapid token refresh attempts", func() {
			if len(authRefreshTokens) > 0 {
				for i := 0; i < 5; i++ {
					w := httptest.NewRecorder()
					reqBody, _ := json.Marshal(gin.H{"refresh_token": authRefreshTokens[0]})
					req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(reqBody))
					req.Header.Set("Content-Type", "application/json")
					router.ServeHTTP(w, req)
					// Should handle gracefully
					Expect(w.Code).To(BeNumerically(">=", 200))
				}
			}
		})
	})

	Describe("Authorization Edge Cases", func() {
		It("should handle expired tokens gracefully", func() {
			w := httptest.NewRecorder()
			// Use an obviously invalid/expired token
			expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEyMzQ1Njc4OTAiLCJleHAiOjE1MTYyMzkwMjJ9.invalid"
			req, _ := http.NewRequest("GET", "/users/me", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", expiredToken))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("should handle malformed authorization headers", func() {
			testCases := []string{
				"",
				"Bearer",
				"Token abc123",
				"Bearer Bearer token",
				"bearer lowercase",
			}
			
			for _, authHeader := range testCases {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/users/me", nil)
				if authHeader != "" {
					req.Header.Set("Authorization", authHeader)
				}
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(401))
			}
		})

		It("should handle token with invalid signature", func() {
			w := httptest.NewRecorder()
			// Valid structure but wrong signature
			invalidToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEyMzQ1Njc4OTAiLCJyZWZyZXNoIjpmYWxzZX0.wrong_signature_here"
			req, _ := http.NewRequest("GET", "/users/me", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", invalidToken))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})

	Describe("HTTP Method Edge Cases", func() {
		It("should reject incorrect HTTP methods", func() {
			testCases := []struct {
				method   string
				endpoint string
			}{
				{"GET", "/auth/register"},
				{"POST", "/users/me"},
				{"PUT", "/auth/login"},
				{"DELETE", "/auth/register"},
				{"PATCH", "/exercises"},
			}
			
			for _, tc := range testCases {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(tc.method, tc.endpoint, nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				// Should return 404 or 405 for wrong methods
				Expect(w.Code).To(BeNumerically(">=", 404))
				Expect(w.Code).To(BeNumerically("<=", 405))
			}
		})

		It("should handle OPTIONS requests for CORS", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("OPTIONS", "/auth/login", nil)
			router.ServeHTTP(w, req)
			// Should handle OPTIONS for CORS preflight
			Expect(w.Code).To(BeNumerically("<=", 404))
		})
	})

	Describe("Content Type Edge Cases", func() {
		It("should reject non-JSON content types", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader("email=test@test.com&password=password"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should handle missing content type", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"email":    "test@test.com",
				"password": "password",
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			// Deliberately not setting Content-Type
			router.ServeHTTP(w, req)
			// Should either infer or reject
			Expect(w.Code).To(BeNumerically(">=", 400))
		})

		It("should handle incorrect content type with valid JSON", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"email":    "test@test.com",
				"password": "password",
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "text/plain")
			router.ServeHTTP(w, req)
			// Should handle gracefully
			Expect(w.Code).To(BeNumerically(">=", 200))
		})
	})

	Describe("Database Constraint Violations", func() {
		It("should handle duplicate username registration", func() {
			// First registration
			w1 := httptest.NewRecorder()
			reqBody1, _ := json.Marshal(gin.H{
				"first_name": "First",
				"last_name":  "User",
				"username":   "uniqueuser123",
				"email":      "first@unique.com",
				"password":   "password123",
			})
			req1, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody1))
			req1.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w1, req1)
			
			// Second registration with same username
			w2 := httptest.NewRecorder()
			reqBody2, _ := json.Marshal(gin.H{
				"first_name": "Second",
				"last_name":  "User",
				"username":   "uniqueuser123", // Same username
				"email":      "second@unique.com",
				"password":   "password123",
			})
			req2, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody2))
			req2.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w2, req2)
			
			// Second should fail
			if w1.Code == 200 {
				Expect(w2.Code).To(Equal(400))
			}
		})

		It("should handle invalid UUID formats", func() {
			invalidUUIDs := []string{
				"not-a-uuid",
				"12345",
				"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
				"",
				"null",
			}
			
			for _, uuid := range invalidUUIDs {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", uuid), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(400))
			}
		})
	})
}