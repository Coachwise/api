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

			// Verify user directly (skip OTP for testing). They also build the
			// exercise this suite trains on, which is coach-only.
			db.Exec("UPDATE users SET status = 'ACTIVE', is_coach = true WHERE id = $1", userID)

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
				"notes":     "Updated session notes",
				"intensity": 7,
				"quality":   4,
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
			Expect(result["intensity"]).To(Equal(float64(7)))
			Expect(result["quality"]).To(Equal(float64(4)))
		})

		It("should complete session and set ended_at timestamp", func() {
			payload := gin.H{
				"status":    "COMPLETED",
				"intensity": 9,
				"quality":   5,
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
			Expect(result["intensity"]).To(Equal(float64(9)))
			Expect(result["quality"]).To(Equal(float64(5)))
		})

		It("should fail with invalid session ID", func() {
			payload := gin.H{
				"notes":     "Test",
				"intensity": 5,
				"quality":   5,
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
				"notes":     "Test",
				"intensity": 5,
				"quality":   5,
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

	Describe("GET /workouts/sessions/analytics/daily - Get Daily Analytics", func() {
		BeforeEach(func() {
			// Create completed sessions with workout logs for analytics
			// Session 1
			session1Payload := gin.H{
				"session_type": "STRENGTH",
				"notes":        "Morning workout",
			}
			session1Body, _ := json.Marshal(session1Payload)
			session1Req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(session1Body))
			session1Req.Header.Set("Content-Type", "application/json")
			session1Req.Header.Set("Authorization", "Bearer "+token)
			session1Resp := httptest.NewRecorder()
			router.ServeHTTP(session1Resp, session1Req)

			var session1Result gin.H
			json.NewDecoder(session1Resp.Body).Decode(&session1Result)
			session1ID := session1Result["id"].(string)

			// Add workout logs to session 1
			for i := 1; i <= 3; i++ {
				logPayload := gin.H{
					"session_id":  session1ID,
					"exercise_id": exerciseID,
					"set_number":  i,
					"reps":        10,
					"weight":      100.0,
					"rpe":         8.0,
					"completed":   true,
				}
				logBody, _ := json.Marshal(logPayload)
				logReq := httptest.NewRequest(http.MethodPost, "/workouts/logs", bytes.NewBuffer(logBody))
				logReq.Header.Set("Content-Type", "application/json")
				logReq.Header.Set("Authorization", "Bearer "+token)
				logResp := httptest.NewRecorder()
				router.ServeHTTP(logResp, logReq)
			}

			// Complete session 1
			completePayload := gin.H{
				"status": "COMPLETED",
			}
			completeBody, _ := json.Marshal(completePayload)
			completeReq := httptest.NewRequest(http.MethodPut, "/workouts/sessions/"+session1ID, bytes.NewBuffer(completeBody))
			completeReq.Header.Set("Content-Type", "application/json")
			completeReq.Header.Set("Authorization", "Bearer "+token)
			completeResp := httptest.NewRecorder()
			router.ServeHTTP(completeResp, completeReq)
		})

		It("should return daily analytics with aggregated data", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/analytics/daily?limit=10", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			// Check pagination structure
			Expect(result["items"]).NotTo(BeNil())
			Expect(result["total"]).NotTo(BeNil())

			items := result["items"].([]interface{})
			Expect(len(items)).To(BeNumerically(">", 0))

			// Verify first analytics item structure
			firstItem := items[0].(map[string]interface{})
			Expect(firstItem["date"]).NotTo(BeNil())
			Expect(firstItem["sessions_count"]).To(BeNumerically(">", 0))
			Expect(firstItem["total_sets"]).NotTo(BeNil())
			Expect(firstItem["exercises_completed"]).NotTo(BeNil())
		})

		It("should return empty items array when no completed sessions exist", func() {
			// Create new user with no sessions
			registerPayload := gin.H{
				"first_name": "New",
				"last_name":  "User",
				"username":   "newanalytics",
				"email":      "newanalytics@test.com",
				"password":   "Password123!",
			}
			registerBody, _ := json.Marshal(registerPayload)
			registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(registerBody))
			registerReq.Header.Set("Content-Type", "application/json")
			registerResp := httptest.NewRecorder()
			router.ServeHTTP(registerResp, registerReq)

			var registerResult gin.H
			json.NewDecoder(registerResp.Body).Decode(&registerResult)
			newUserID := registerResult["id"].(string)

			// Activate user
			db.Exec("UPDATE users SET status = 'ACTIVE' WHERE id = $1", newUserID)

			// Login
			loginPayload := gin.H{
				"username": "newanalytics@test.com",
				"password": "Password123!",
			}
			loginBody, _ := json.Marshal(loginPayload)
			loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(loginBody))
			loginReq.Header.Set("Content-Type", "application/json")
			loginResp := httptest.NewRecorder()
			router.ServeHTTP(loginResp, loginReq)

			var loginResult gin.H
			json.NewDecoder(loginResp.Body).Decode(&loginResult)
			newToken := loginResult["token"].(string)

			// Get analytics for user with no sessions
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/analytics/daily", nil)
			req.Header.Set("Authorization", "Bearer "+newToken)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			items := result["items"].([]interface{})
			Expect(len(items)).To(Equal(0))
			Expect(result["total"]).To(Equal(float64(0)))
		})

		It("should support pagination parameters", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/analytics/daily?limit=5&offset=0", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			Expect(result["items"]).NotTo(BeNil())
			Expect(result["total"]).NotTo(BeNil())
		})

		It("should fail without authentication", func() {
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/analytics/daily", nil)

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusUnauthorized))
		})

		It("should aggregate multiple sessions on the same day", func() {
			// Create second session for the same day
			session2Payload := gin.H{
				"session_type": "STRENGTH",
				"notes":        "Evening workout",
			}
			session2Body, _ := json.Marshal(session2Payload)
			session2Req := httptest.NewRequest(http.MethodPost, "/workouts/sessions", bytes.NewBuffer(session2Body))
			session2Req.Header.Set("Content-Type", "application/json")
			session2Req.Header.Set("Authorization", "Bearer "+token)
			session2Resp := httptest.NewRecorder()
			router.ServeHTTP(session2Resp, session2Req)

			var session2Result gin.H
			json.NewDecoder(session2Resp.Body).Decode(&session2Result)
			session2ID := session2Result["id"].(string)

			// Add workout logs to session 2
			for i := 1; i <= 2; i++ {
				logPayload := gin.H{
					"session_id":  session2ID,
					"exercise_id": exerciseID,
					"set_number":  i,
					"reps":        12,
					"weight":      80.0,
					"completed":   true,
				}
				logBody, _ := json.Marshal(logPayload)
				logReq := httptest.NewRequest(http.MethodPost, "/workouts/logs", bytes.NewBuffer(logBody))
				logReq.Header.Set("Content-Type", "application/json")
				logReq.Header.Set("Authorization", "Bearer "+token)
				logResp := httptest.NewRecorder()
				router.ServeHTTP(logResp, logReq)
			}

			// Complete session 2
			completePayload := gin.H{
				"status": "COMPLETED",
			}
			completeBody, _ := json.Marshal(completePayload)
			completeReq := httptest.NewRequest(http.MethodPut, "/workouts/sessions/"+session2ID, bytes.NewBuffer(completeBody))
			completeReq.Header.Set("Content-Type", "application/json")
			completeReq.Header.Set("Authorization", "Bearer "+token)
			completeResp := httptest.NewRecorder()
			router.ServeHTTP(completeResp, completeReq)

			// Get analytics
			req := httptest.NewRequest(http.MethodGet, "/workouts/sessions/analytics/daily?limit=10", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)

			items := result["items"].([]interface{})
			Expect(len(items)).To(BeNumerically(">", 0))

			// Find today's analytics
			todayItem := items[0].(map[string]interface{})

			// Should have 2 sessions (from BeforeEach + this test)
			Expect(todayItem["sessions_count"]).To(BeNumerically(">=", 2))

			// Should have aggregated sets (3 from session 1 + 2 from session 2 = 5)
			Expect(todayItem["total_sets"]).To(BeNumerically(">=", 5))
		})
	})
}
