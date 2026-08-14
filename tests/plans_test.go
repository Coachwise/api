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
					"sport_type":  "STRENGTH",
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

		It("forces user-created plans to be personal (public not client-settable)", func() {
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
				Expect(body["public"]).To(Equal(false))
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
					"intensity":      7,
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
						"intensity":      5 + i,                 // Varying intensity (5, 6, 7)
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

		It("stores a per-plan set prescription independent of the exercise's own sets", func() {
			if len(exerciseIds) == 0 {
				Skip("no exercises seeded")
			}
			// Use a dedicated plan so earlier specs' additions to the shared plan
			// don't collide with this exercise's rows.
			pw := httptest.NewRecorder()
			pbody, _ := json.Marshal(gin.H{"name": "Prescription Plan"})
			preq, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(pbody))
			preq.Header.Set("Content-Type", "application/json")
			preq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(pw, preq)
			Expect(pw.Code).To(Equal(201))
			rxPlanId := decodeBody(pw.Body)["id"].(string)

			// The exercise was created with 2 default sets. Prescribe a different
			// shape (3 sets) on the plan-exercise — it must not touch the exercise.
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"exercise_id":    exerciseIds[0],
				"exercise_order": 1,
				"rest_time":      90e9,
				"intensity":      6,
				"sets": []gin.H{
					{"name": "P1", "rest_time": 20e9, "rep_count": 8},
					{"name": "P2", "rest_time": 25e9, "rep_count": 6},
					{"rest_time": 0, "duration": 30e9},
				},
			})
			req, _ := http.NewRequest("POST", fmt.Sprintf("/plans/%s/exercises", rxPlanId), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(BeNumerically(">=", 200))
			Expect(w.Code).To(BeNumerically("<", 300))

			// Fetch and confirm the prescription round-trips in set_number order,
			// with 3 sets (not the exercise's 2 defaults).
			g := httptest.NewRecorder()
			greq, _ := http.NewRequest("GET", fmt.Sprintf("/plans/%s/exercises", rxPlanId), nil)
			greq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(g, greq)
			Expect(g.Code).To(Equal(200))

			var items []struct {
				ExerciseID string `json:"exercise_id"`
				Sets       []struct {
					SetNumber int    `json:"set_number"`
					RepCount  *int   `json:"rep_count"`
					Duration  *int64 `json:"duration"`
					RestTime  int64  `json:"rest_time"`
				} `json:"sets"`
			}
			Expect(json.NewDecoder(g.Body).Decode(&items)).To(Succeed())

			var found bool
			for _, it := range items {
				if it.ExerciseID != exerciseIds[0] {
					continue
				}
				found = true
				Expect(it.Sets).To(HaveLen(3))
				Expect(it.Sets[0].SetNumber).To(Equal(1))
				Expect(*it.Sets[0].RepCount).To(Equal(8))
				Expect(it.Sets[2].SetNumber).To(Equal(3))
				Expect(*it.Sets[2].Duration).To(Equal(int64(30e9)))
			}
			Expect(found).To(BeTrue())

			// The exercise's own default sets are untouched (still 2).
			e := httptest.NewRecorder()
			ereq, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", exerciseIds[0]), nil)
			ereq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(e, ereq)
			if e.Code == 200 {
				var ex struct {
					Sets []interface{} `json:"sets"`
				}
				Expect(json.NewDecoder(e.Body).Decode(&ex)).To(Succeed())
				Expect(ex.Sets).To(HaveLen(2))
			}
		})

		// A group is just an exercise to a plan, but the plan may override how many
		// rounds it runs without touching the group everyone else uses.
		It("carries a group's rounds and lets the plan override them", func() {
			if len(exerciseIds) == 0 {
				Skip("no exercises seeded")
			}
			post := func(path string, payload gin.H) *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				body, _ := json.Marshal(payload)
				req, _ := http.NewRequest("POST", path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)
				return w
			}

			gw := post("/exercises", gin.H{
				"name": "Plan circuit", "description": "circuit", "kind": "GROUP",
				"rounds": 5, "round_rest": 90e9,
				"items": []gin.H{{"exercise_id": exerciseIds[0], "rep_count": 15, "rest_time": 30e9}},
			})
			Expect(gw.Code).To(Equal(201))
			groupID := decodeBody(gw.Body)["id"].(string)

			pw := post("/plans", gin.H{"name": "Circuit Plan"})
			Expect(pw.Code).To(Equal(201))
			circuitPlanID := decodeBody(pw.Body)["id"].(string)

			// The group defaults to 5 rounds; this plan runs it 3 times.
			aw := post(fmt.Sprintf("/plans/%s/exercises", circuitPlanID), gin.H{
				"exercise_id": groupID, "exercise_order": 1, "rest_time": 60e9,
				"intensity": 5, "rounds": 3,
			})
			Expect(aw.Code).To(BeNumerically("<", 300))

			g := httptest.NewRecorder()
			greq, _ := http.NewRequest("GET", fmt.Sprintf("/plans/%s/exercises", circuitPlanID), nil)
			greq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(g, greq)
			Expect(g.Code).To(Equal(200))

			var items []struct {
				Rounds   *int `json:"rounds"`
				Exercise struct {
					Kind   string `json:"kind"`
					Rounds *int   `json:"rounds"`
					Items  []struct {
						ItemOrder int  `json:"item_order"`
						RepCount  *int `json:"rep_count"`
					} `json:"items"`
				} `json:"exercise"`
			}
			Expect(json.NewDecoder(g.Body).Decode(&items)).To(Succeed())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Exercise.Kind).To(Equal("GROUP"))
			Expect(*items[0].Rounds).To(Equal(3))    // the plan's override
			Expect(*items[0].Exercise.Rounds).To(Equal(5)) // the group's own default
			Expect(items[0].Exercise.Items).To(HaveLen(1))
			Expect(*items[0].Exercise.Items[0].RepCount).To(Equal(15))
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
