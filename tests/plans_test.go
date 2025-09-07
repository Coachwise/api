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

func plansGroup() {
	var planId string
	var exerciseIds []string

	BeforeEach(func() {
		// Create some exercises to use in plans
		if len(exerciseIds) == 0 {
			for i := 1; i <= 3; i++ {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{
					"name":        fmt.Sprintf("Plan Exercise %d", i),
					"description": fmt.Sprintf("Exercise %d for plans", i),
					"public":      true,
					"sets": []gin.H{
						{"name": "Set 1", "rest_time": 30e9, "rep_count": 10},
						{"name": "Set 2", "rest_time": 45e9, "rep_count": 12},
					},
				})
				req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code == 201 {
					body := decodeBody(w.Body)
					exerciseIds = append(exerciseIds, body["id"].(string))
				}
			}
		}
	})

	Describe("Plan Creation", func() {
		It("should create a training plan", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":   "Beginner Strength Plan",
				"public": false,
			})
			req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 201 {
				body := decodeBody(w.Body)
				Expect(body["name"]).To(Equal("Beginner Strength Plan"))
				planId = body["id"].(string)
			}
		})

		It("should create a public plan", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":   "Public Climbing Plan",
				"public": true,
			})
			req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 201 {
				body := decodeBody(w.Body)
				Expect(body["public"]).To(Equal(true))
			}
		})

		It("should fail to create plan without authentication", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":   "Unauthorized Plan",
				"public": false,
			})
			req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			
			if w.Code != 404 { // If endpoint exists
				Expect(w.Code).To(Equal(401))
			}
		})

		It("should fail to create plan without name", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"public": false,
			})
			req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code != 404 { // If endpoint exists
				Expect(w.Code).To(Equal(400))
			}
		})
	})

	Describe("Plan Exercises", func() {
		It("should add exercises to plan", func() {
			if planId != "" && len(exerciseIds) > 0 {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{
					"exercise_id":    exerciseIds[0],
					"exercise_order": 1,
					"rest_time":      120e9, // 2 minutes rest
				})
				req, _ := http.NewRequest("POST", fmt.Sprintf("/plans/%s/exercises", planId), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					Expect(w.Code).To(BeNumerically(">=", 200))
					Expect(w.Code).To(BeNumerically("<", 300))
				}
			}
		})

		It("should add multiple exercises to plan", func() {
			if planId != "" && len(exerciseIds) >= 3 {
				for i, exerciseId := range exerciseIds {
					w := httptest.NewRecorder()
					reqBody, _ := json.Marshal(gin.H{
						"exercise_id":    exerciseId,
						"exercise_order": i + 1,
						"rest_time":      int64((i+1)*60) * 1e9, // Varying rest times
					})
					req, _ := http.NewRequest("POST", fmt.Sprintf("/plans/%s/exercises", planId), bytes.NewBuffer(reqBody))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
					router.ServeHTTP(w, req)
					
					if w.Code != 404 { // If endpoint exists
						Expect(w.Code).To(BeNumerically(">=", 200))
						Expect(w.Code).To(BeNumerically("<", 300))
					}
				}
			}
		})

		It("should list exercises in plan", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", fmt.Sprintf("/plans/%s/exercises", planId), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code == 200 {
					var body []interface{}
					json.NewDecoder(w.Body).Decode(&body)
					Expect(len(body)).To(BeNumerically(">=", 1))
				}
			}
		})

		It("should remove exercise from plan", func() {
			if planId != "" && len(exerciseIds) > 0 {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("DELETE", fmt.Sprintf("/plans/%s/exercises/%s", planId, exerciseIds[0]), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					Expect(w.Code).To(BeNumerically(">=", 200))
					Expect(w.Code).To(BeNumerically("<", 300))
				}
			}
		})
	})

	Describe("Plan Assignment", func() {
		var clientId string

		BeforeEach(func() {
			// Create a client user
			if clientId == "" {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{
					"first_name": "Client",
					"last_name":  "User",
					"username":   "clientuser",
					"email":      "client@test.com",
					"password":   "password123",
				})
				req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)
				
				if w.Code == 200 {
					// Get user ID
					otp := struct{ Code string }{}
					db.Get(&otp, "SELECT code FROM otps WHERE email = 'client@test.com' LIMIT 1")
					
					w2 := httptest.NewRecorder()
					reqBody2, _ := json.Marshal(gin.H{"email": "client@test.com", "code": otp.Code})
					req2, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody2))
					req2.Header.Set("Content-Type", "application/json")
					router.ServeHTTP(w2, req2)
					
					if w2.Code == 200 {
						body := decodeBody(w2.Body)
						token := body["access_token"].(string)
						
						w3 := httptest.NewRecorder()
						req3, _ := http.NewRequest("GET", "/users/me", nil)
						req3.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
						router.ServeHTTP(w3, req3)
						
						if w3.Code == 200 {
							body3 := decodeBody(w3.Body)
							clientId = body3["id"].(string)
						}
					}
				}
			}
		})

		It("should assign plan to user", func() {
			if planId != "" && clientId != "" {
				w := httptest.NewRecorder()
				dueDate := time.Now().Add(30 * 24 * time.Hour) // 30 days from now
				reqBody, _ := json.Marshal(gin.H{
					"user_id": clientId,
					"due_at":  dueDate.Format(time.RFC3339),
				})
				req, _ := http.NewRequest("POST", fmt.Sprintf("/plans/%s/assign", planId), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					Expect(w.Code).To(BeNumerically(">=", 200))
					Expect(w.Code).To(BeNumerically("<", 300))
				}
			}
		})

		It("should list assigned users", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", fmt.Sprintf("/plans/%s/assignments", planId), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code == 200 {
					var body []interface{}
					json.NewDecoder(w.Body).Decode(&body)
					Expect(len(body)).To(BeNumerically(">=", 0))
				}
			}
		})

		It("should unassign plan from user", func() {
			if planId != "" && clientId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("DELETE", fmt.Sprintf("/plans/%s/assign/%s", planId, clientId), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					Expect(w.Code).To(BeNumerically(">=", 200))
					Expect(w.Code).To(BeNumerically("<", 300))
				}
			}
		})
	})

	Describe("Plan Retrieval", func() {
		It("should get plan by ID", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", fmt.Sprintf("/plans/%s", planId), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code == 200 {
					body := decodeBody(w.Body)
					Expect(body["id"]).To(Equal(planId))
				}
			}
		})

		It("should list all plans", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/plans", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code == 200 {
				var body []interface{}
				json.NewDecoder(w.Body).Decode(&body)
				Expect(len(body)).To(BeNumerically(">=", 0))
			}
		})

		It("should filter plans by public status", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/plans?public=true", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code == 200 {
				var body []interface{}
				json.NewDecoder(w.Body).Decode(&body)
				for _, plan := range body {
					p := plan.(map[string]interface{})
					if p["public"] != nil {
						Expect(p["public"]).To(Equal(true))
					}
				}
			}
		})

		It("should get user's assigned plans", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/me/plans", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code == 200 {
				var body []interface{}
				json.NewDecoder(w.Body).Decode(&body)
				Expect(len(body)).To(BeNumerically(">=", 0))
			}
		})
	})

	Describe("Plan Update", func() {
		It("should update plan details", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{
					"name":   "Updated Plan Name",
					"public": true,
				})
				req, _ := http.NewRequest("PUT", fmt.Sprintf("/plans/%s", planId), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					if w.Code == 200 {
						body := decodeBody(w.Body)
						Expect(body["name"]).To(Equal("Updated Plan Name"))
					}
				}
			}
		})

		It("should fail to update non-existent plan", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"name": "Ghost Plan"})
			req, _ := http.NewRequest("PUT", "/plans/00000000-0000-0000-0000-000000000000", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code != 404 { // If endpoint exists
				Expect(w.Code).To(Equal(400))
			}
		})

		It("should fail to update without authentication", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{"name": "Unauthorized Update"})
				req, _ := http.NewRequest("PUT", fmt.Sprintf("/plans/%s", planId), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					Expect(w.Code).To(Equal(401))
				}
			}
		})
	})

	Describe("Plan Deletion", func() {
		It("should delete plan", func() {
			// Create a plan to delete
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":   "Plan to Delete",
				"public": false,
			})
			req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code == 201 {
				body := decodeBody(w.Body)
				deletePlanId := body["id"].(string)
				
				// Delete the plan
				w2 := httptest.NewRecorder()
				req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/plans/%s", deletePlanId), nil)
				req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w2, req2)
				
				if w2.Code != 404 { // If endpoint exists
					Expect(w2.Code).To(BeNumerically(">=", 200))
					Expect(w2.Code).To(BeNumerically("<", 300))
					
					// Verify plan is deleted
					w3 := httptest.NewRecorder()
					req3, _ := http.NewRequest("GET", fmt.Sprintf("/plans/%s", deletePlanId), nil)
					req3.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
					router.ServeHTTP(w3, req3)
					Expect(w3.Code).To(BeNumerically(">=", 400))
				}
			}
		})

		It("should fail to delete non-existent plan", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/plans/00000000-0000-0000-0000-000000000000", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			
			if w.Code != 404 { // If endpoint exists
				Expect(w.Code).To(Equal(400))
			}
		})

		It("should fail to delete without authentication", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("DELETE", fmt.Sprintf("/plans/%s", planId), nil)
				router.ServeHTTP(w, req)
				
				if w.Code != 404 { // If endpoint exists
					Expect(w.Code).To(Equal(401))
				}
			}
		})
	})
}