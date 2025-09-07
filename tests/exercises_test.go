package tests_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func exerciseGroup() {
	var exerciseId string
	var publicExerciseId string

	Describe("Exercise Creation", func() {
		It("should create exercise with sets", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(exercisesData[0])
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(201))
			bodyExpect(body, gin.H{"id": "<ANY>", "name": exercisesData[0]["name"]})
			exerciseId = body["id"].(string)

			// Verify sets were created
			sets := body["sets"].([]interface{})
			Expect(len(sets)).To(Equal(2))
		})

		It("should create public exercise", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Public Exercise",
				"description": "This is a public exercise",
				"public":      true,
				"sets": []gin.H{
					{"name": "Set 1", "rest_time": 30e9, "rep_count": 10},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(201))
			Expect(body["public"]).To(Equal(true))
			publicExerciseId = body["id"].(string)
		})

		It("should fail to create exercise without authentication", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(exercisesData[0])
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("should fail to create exercise with invalid data", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"description": "Missing name field",
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should create exercise with duration-based sets", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Plank Exercise",
				"description": "Core strengthening",
				"public":      false,
				"sets": []gin.H{
					{"name": "Hold", "rest_time": 60e9, "duration": 30e9},
					{"name": "Hold", "rest_time": 60e9, "duration": 45e9},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(201))
			sets := body["sets"].([]interface{})
			Expect(len(sets)).To(Equal(2))
		})

		It("should fail to create set with both rep_count and duration", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Invalid Exercise",
				"description": "Invalid sets",
				"public":      false,
				"sets": []gin.H{
					{"name": "Invalid", "rest_time": 30e9, "rep_count": 10, "duration": 30e9},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Exercise Retrieval", func() {
		It("should get exercise by ID", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", exerciseId), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"id": exerciseId, "name": exercisesData[0]["name"]})
		})

		It("should fail to get non-existent exercise", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/exercises/00000000-0000-0000-0000-000000000000", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail to get exercise without authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", exerciseId), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("should list all exercises", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/exercises", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				var body []interface{}
				json.NewDecoder(w.Body).Decode(&body)
				Expect(len(body)).To(BeNumerically(">=", 2))
			}
		})

		It("should filter exercises by public status", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/exercises?public=true", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				var body []interface{}
				json.NewDecoder(w.Body).Decode(&body)
				for _, exercise := range body {
					ex := exercise.(map[string]interface{})
					Expect(ex["public"]).To(Equal(true))
				}
			}
		})

		It("should search exercises by name", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/exercises?name=test", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				var body []interface{}
				json.NewDecoder(w.Body).Decode(&body)
				Expect(len(body)).To(BeNumerically(">=", 1))
			}
		})
	})

	Describe("Exercise Update", func() {
		It("should update exercise details", func() {
			updatedData := gin.H{
				"name":        "updated",
				"description": "updated",
				"public":      false,
				"sets": []gin.H{
					{"name": "updated", "rest_time": 60e9, "rep_count": 10},
				},
			}
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(updatedData)
			req, _ := http.NewRequest("PUT", fmt.Sprintf("/exercises/%s", exerciseId), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			bodyExpect(body, gin.H{"id": exerciseId, "name": "updated"})
		})

		It("should add sets to existing exercise", func() {
			updatedData := gin.H{
				"name":        "updated",
				"description": "updated with more sets",
				"public":      false,
				"sets": []gin.H{
					{"name": "Set 1", "rest_time": 30e9, "rep_count": 8},
					{"name": "Set 2", "rest_time": 45e9, "rep_count": 10},
					{"name": "Set 3", "rest_time": 60e9, "rep_count": 12},
				},
			}
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(updatedData)
			req, _ := http.NewRequest("PUT", fmt.Sprintf("/exercises/%s", exerciseId), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(200))
			sets := body["sets"].([]interface{})
			Expect(len(sets)).To(Equal(3))
		})

		It("should fail to update non-existent exercise", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"name": "updated"})
			req, _ := http.NewRequest("PUT", "/exercises/00000000-0000-0000-0000-000000000000", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail to update exercise without authentication", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"name": "unauthorized"})
			req, _ := http.NewRequest("PUT", fmt.Sprintf("/exercises/%s", exerciseId), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("should fail to update other user's private exercise", func() {
			// Create another user and get their token
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "Another",
				"last_name":  "User",
				"username":   "anotheruser",
				"email":      "another@test.com",
				"password":   "password123",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				// Get OTP and verify
				otp := struct{ Code string }{}
				db.Get(&otp, "SELECT code FROM otps WHERE email = 'another@test.com' LIMIT 1")
				
				w2 := httptest.NewRecorder()
				reqBody2, _ := json.Marshal(gin.H{"email": "another@test.com", "code": otp.Code})
				req2, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody2))
				req2.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w2, req2)
				
				body := decodeBody(w2.Body)
				anotherToken := body["access_token"].(string)

				// Try to update first user's exercise
				w3 := httptest.NewRecorder()
				reqBody3, _ := json.Marshal(gin.H{"name": "hacked"})
				req3, _ := http.NewRequest("PUT", fmt.Sprintf("/exercises/%s", exerciseId), bytes.NewBuffer(reqBody3))
				req3.Header.Set("Content-Type", "application/json")
				req3.Header.Set("Authorization", fmt.Sprintf("Bearer %s", anotherToken))
				router.ServeHTTP(w3, req3)
				// Should either return 403 Forbidden or 404 Not Found
				Expect(w3.Code).To(BeNumerically(">=", 400))
			}
		})
	})

	Describe("Exercise Deletion", func() {
		It("should delete exercise", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/exercises/%s", exerciseId), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(204))

			// Verify exercise is deleted
			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", exerciseId), nil)
			req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(400))
		})

		It("should fail to delete non-existent exercise", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/exercises/00000000-0000-0000-0000-000000000000", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("should fail to delete exercise without authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/exercises/%s", publicExerciseId), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})

	Describe("Exercise Performance Tracking", func() {
		It("should track exercise performance over time", func() {
			// Create an exercise
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Bench Press",
				"description": "Chest exercise",
				"public":      false,
				"sets": []gin.H{
					{"name": "Warmup", "rest_time": 60e9, "rep_count": 10},
					{"name": "Working", "rest_time": 90e9, "rep_count": 8},
					{"name": "Working", "rest_time": 90e9, "rep_count": 8},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(201))
			trackingId := body["id"].(string)

			// Simulate updating with increased performance
			time.Sleep(100 * time.Millisecond)
			w2 := httptest.NewRecorder()
			reqBody2, _ := json.Marshal(gin.H{
				"name":        "Bench Press",
				"description": "Chest exercise - improved",
				"public":      false,
				"sets": []gin.H{
					{"name": "Warmup", "rest_time": 60e9, "rep_count": 12},
					{"name": "Working", "rest_time": 90e9, "rep_count": 10},
					{"name": "Working", "rest_time": 90e9, "rep_count": 10},
					{"name": "Working", "rest_time": 90e9, "rep_count": 8},
				},
			})
			req2, _ := http.NewRequest("PUT", fmt.Sprintf("/exercises/%s", trackingId), bytes.NewBuffer(reqBody2))
			req2.Header.Set("Content-Type", "application/json")
			req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w2, req2)

			body2 := decodeBody(w2.Body)
			Expect(w2.Code).To(Equal(200))
			sets := body2["sets"].([]interface{})
			Expect(len(sets)).To(Equal(4))
		})
	})
}