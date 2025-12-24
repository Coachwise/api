package tests_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var sessionsGroup = func() {
	var (
		token      string
		sessionID  string
		exerciseID string
	)

	BeforeEach(func() {
		// Create and login user to get token
		if token == "" {
			// Register user
			registerPayload := gin.H{
				"first_name": "Session",
				"last_name":  "Tester",
				"username":   "sessiontester",
				"email":      "sessiontest@test.com",
				"password":   "Password123!",
			}
			registerBody, _ := json.Marshal(registerPayload)
			registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(registerBody))
			registerReq.Header.Set("Content-Type", "application/json")
			registerResp := httptest.NewRecorder()
			router.ServeHTTP(registerResp, registerReq)
			Expect(registerResp.Code).To(Equal(http.StatusOK))

			var registerResult gin.H
			json.NewDecoder(registerResp.Body).Decode(&registerResult)
			userID := registerResult["id"].(string)

			// Verify user directly (skip OTP for testing)
			db.Exec("UPDATE users SET status = 'ACTIVE' WHERE id = $1", userID)

			// Login
			loginPayload := gin.H{
				"username": "sessiontest@test.com",
				"password": "Password123!",
			}
			loginBody, _ := json.Marshal(loginPayload)
			loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(loginBody))
			loginReq.Header.Set("Content-Type", "application/json")
			loginResp := httptest.NewRecorder()
			router.ServeHTTP(loginResp, loginReq)
			Expect(loginResp.Code).To(Equal(http.StatusOK))

			var loginResult gin.H
			json.NewDecoder(loginResp.Body).Decode(&loginResult)
			token = loginResult["token"].(string)

			// Create an exercise for workout logs
			exercisePayload := gin.H{
				"name":        "Bench Press",
				"description": "Chest exercise",
				"sport_type":  "STRENGTH",
			}
			exerciseBody, _ := json.Marshal(exercisePayload)
			exerciseReq := httptest.NewRequest(http.MethodPost, "/exercises", bytes.NewBuffer(exerciseBody))
			exerciseReq.Header.Set("Content-Type", "application/json")
			exerciseReq.Header.Set("Authorization", "Bearer "+token)
			exerciseResp := httptest.NewRecorder()
			router.ServeHTTP(exerciseResp, exerciseReq)
			Expect(exerciseResp.Code).To(Equal(http.StatusOK))

			var exerciseResult gin.H
			json.NewDecoder(exerciseResp.Body).Decode(&exerciseResult)
			exerciseID = exerciseResult["id"].(string)
		}
	})

	Describe("POST /workouts/sessions - Create Session", func() {
		It("should create a new workout session successfully", func() {
			payload := gin.H{
				"session_type": "STRENGTH",
				"notes":        "Test workout session",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(result["session_type"]).To(Equal("STRENGTH"))
			Expect(result["status"]).To(Equal("ACTIVE"))
			Expect(result["notes"]).To(Equal("Test workout session"))
			Expect(result["id"]).NotTo(BeNil())

			// Save session ID for subsequent tests
			sessionID = result["id"].(string)
		})

		It("should fail without authentication", func() {
			payload := gin.H{
				"session_type": "STRENGTH",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})

		It("should fail with missing session_type", func() {
			payload := gin.H{
				"notes": "Missing session type",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GET /workouts/sessions/active - Get Active Sessions", func() {
		BeforeEach(func() {
			// Create a session for testing
			if sessionID == "" {
				payload := gin.H{
					"session_type": "STRENGTH",
					"notes":        "Active session test",
				}
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				var result gin.H
				json.NewDecoder(resp.Body).Decode(&result)
				sessionID = result["id"].(string)
			}
		})

		It("should return active sessions for the user", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/active", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result []gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(len(result)).To(BeNumerically(">", 0))
			Expect(result[0]["status"]).To(Equal("ACTIVE"))
		})

		It("should fail without authentication", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/active", nil)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("GET /workouts/sessions/:id - Get Session By ID", func() {
		BeforeEach(func() {
			// Ensure we have a session
			if sessionID == "" {
				payload := gin.H{
					"session_type": "CLIMBING",
					"notes":        "Get by ID test",
				}
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				var result gin.H
				json.NewDecoder(resp.Body).Decode(&result)
				sessionID = result["id"].(string)
			}
		})

		It("should get a session by ID successfully", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/"+sessionID, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(result["id"]).To(Equal(sessionID))
			Expect(result["status"]).To(Equal("ACTIVE"))
		})

		It("should fail with invalid session ID", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/invalid-id", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("should fail with non-existent session ID", func() {
			fakeID := "00000000-0000-0000-0000-000000000000"
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/"+fakeID, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusNotFound))
		})

		It("should fail without authentication", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/"+sessionID, nil)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("PUT /workouts/sessions/:id - Update Session", func() {
		BeforeEach(func() {
			// Ensure we have a session
			if sessionID == "" {
				payload := gin.H{
					"session_type": "STRENGTH",
					"notes":        "Update test session",
				}
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				var result gin.H
				json.NewDecoder(resp.Body).Decode(&result)
				sessionID = result["id"].(string)
			}
		})

		It("should update session notes successfully", func() {
			payload := gin.H{
				"notes": "Updated session notes",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/workouts/sessions/"+sessionID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(result["notes"]).To(Equal("Updated session notes"))
		})

		It("should complete session and set ended_at timestamp", func() {
			payload := gin.H{
				"status": "COMPLETED",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/workouts/sessions/"+sessionID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(result["status"]).To(Equal("COMPLETED"))
			Expect(result["ended_at"]).NotTo(BeNil())
		})

		It("should fail with invalid session ID", func() {
			payload := gin.H{
				"notes": "Test",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/workouts/sessions/invalid-id", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("should fail without authentication", func() {
			payload := gin.H{
				"notes": "Test",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/workouts/sessions/"+sessionID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("GET /workouts/sessions/:id/logs - Get Session Workout Logs", func() {
		var logSessionID string

		BeforeEach(func() {
			// Create a fresh session for logs
			payload := gin.H{
				"session_type": "STRENGTH",
				"notes":        "Session for logs",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)
			logSessionID = result["id"].(string)

			// Add a workout log
			logPayload := gin.H{
				"session_id":   logSessionID,
				"exercise_id":  exerciseID,
				"set_number":   1,
				"reps":         10,
				"weight":       100.0,
				"rpe":          8.0,
				"completed":    true,
			}
			logBody, _ := json.Marshal(logPayload)
			logReq := httptest.NewRequest(http.MethodPost, "/workouts/logs", bytes.NewBuffer(logBody))
			logReq.Header.Set("Content-Type", "application/json")
			logReq.Header.Set("Authorization", "Bearer "+token)
			logResp := httptest.NewRecorder()
			router.ServeHTTP(logResp, logReq)
		})

		It("should get all workout logs for a session", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/"+logSessionID+"/logs", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result []gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(len(result)).To(BeNumerically(">", 0))
			Expect(result[0]["session_id"]).To(Equal(logSessionID))
		})

		It("should return empty array for session with no logs", func() {
			// Create new session without logs
			payload := gin.H{
				"session_type": "CLIMBING",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			var createResult gin.H
			json.NewDecoder(resp.Body).Decode(&createResult)
			emptySessionID := createResult["id"].(string)

			// Get logs for empty session
			logsReq := httptest.NewRequest(http.MethodGet, "/workouts/sessions/"+emptySessionID+"/logs", nil)
			logsReq.Header.Set("Authorization", "Bearer "+token)
			logsResp := httptest.NewRecorder()
			router.ServeHTTP(logsResp, logsReq)

			Expect(logsResp.Code).To(Equal(http.StatusOK))

			var result []gin.H
			json.NewDecoder(logsResp.Body).Decode(&result)
			Expect(len(result)).To(Equal(0))
		})

		It("should fail without authentication", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/"+logSessionID+"/logs", nil)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})
	})
}
