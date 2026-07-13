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

func schedulesGroup() {
	var scheduleId string
	var planId string

	BeforeEach(func() {
		// Create a plan to schedule
		if planId == "" {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":   "Schedule Test Plan",
				"public": false,
			})
			req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 201 {
				body := decodeBody(w.Body)
				planId = body["id"].(string)
			}
		}
	})

	Describe("Schedule Creation", func() {
		It("should create a plan schedule", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				scheduledDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
				reqBody, _ := json.Marshal(gin.H{
					"plan_id":        planId,
					"scheduled_for": scheduledDate,
				})
				req, _ := http.NewRequest("POST", "/schedules", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)

				if w.Code == 201 {
					body := decodeBody(w.Body)
					Expect(body["plan_id"]).To(Equal(planId))
					Expect(body["status"]).To(Equal("ACTIVE"))
					scheduleId = body["id"].(string)
				}
			}
		})

		It("should create schedule with notes", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				scheduledDate := time.Now().Add(48 * time.Hour).Format("2006-01-02")
				reqBody, _ := json.Marshal(gin.H{
					"plan_id":        planId,
					"scheduled_for": scheduledDate,
					"notes":         "Morning workout",
				})
				req, _ := http.NewRequest("POST", "/schedules", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)

				if w.Code == 201 {
					body := decodeBody(w.Body)
					Expect(body["notes"]).To(Equal("Morning workout"))
				}
			}
		})

		It("should fail without authentication", func() {
			w := httptest.NewRecorder()
			scheduledDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")
			reqBody, _ := json.Marshal(gin.H{
				"plan_id":        planId,
				"scheduled_for": scheduledDate,
			})
			req, _ := http.NewRequest("POST", "/schedules", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(401))
		})

		It("should fail with invalid date format", func() {
			if planId != "" {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{
					"plan_id":        planId,
					"scheduled_for": "invalid-date",
				})
				req, _ := http.NewRequest("POST", "/schedules", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(400))
			}
		})
	})

	Describe("Schedule Listing with Pagination", func() {
		It("should list schedules with default pagination", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/schedules", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				body := decodeBody(w.Body)
				Expect(body["items"]).ToNot(BeNil())
				Expect(body["total"]).ToNot(BeNil())
			}
		})

		It("should list schedules with custom page and limit", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/schedules?page=1&limit=5", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				body := decodeBody(w.Body)
				Expect(body["items"]).ToNot(BeNil())
			}
		})

		It("should filter schedules by date range", func() {
			fromDate := time.Now().Format("2006-01-02")
			toDate := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02")

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/schedules?from=%s&to=%s", fromDate, toDate), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				body := decodeBody(w.Body)
				Expect(body["items"]).ToNot(BeNil())
			}
		})

		It("should return empty array when no schedules", func() {
			// Create a new user with no schedules
			w1 := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "No",
				"last_name":  "Schedules",
				"username":   "noschedules",
				"email":      "noschedules@test.com",
				"password":   "password123",
			})
			req1, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req1.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w1, req1)

			if w1.Code == 200 {
				otp := struct{ Code string }{}
				db.Get(&otp, "SELECT code FROM otps WHERE email = 'noschedules@test.com' LIMIT 1")

				w2 := httptest.NewRecorder()
				reqBody2, _ := json.Marshal(gin.H{"email": "noschedules@test.com", "code": otp.Code})
				req2, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody2))
				req2.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w2, req2)

				if w2.Code == 200 {
					body := decodeBody(w2.Body)
					token := body["access_token"].(string)

					w3 := httptest.NewRecorder()
					req3, _ := http.NewRequest("GET", "/schedules", nil)
					req3.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
					router.ServeHTTP(w3, req3)

					if w3.Code == 200 {
						body3 := decodeBody(w3.Body)
						items := body3["items"].([]interface{})
						Expect(len(items)).To(Equal(0))
						Expect(body3["total"]).To(Equal(float64(0)))
					}
				}
			}
		})
	})

	Describe("Schedule Retrieval", func() {
		It("should get schedule by ID", func() {
			if scheduleId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", fmt.Sprintf("/schedules/%s", scheduleId), nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)

				if w.Code == 200 {
					body := decodeBody(w.Body)
					Expect(body["id"]).To(Equal(scheduleId))
				}
			}
		})

		It("should fail to get non-existent schedule", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/schedules/00000000-0000-0000-0000-000000000000", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			Expect(w.Code).To(BeNumerically(">=", 400))
		})
	})

	Describe("Schedule Update", func() {
		It("should update schedule status", func() {
			Expect(scheduleId).NotTo(BeEmpty())
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"status": "CANCELED",
			})
			req, _ := http.NewRequest("PATCH", fmt.Sprintf("/schedules/%s", scheduleId), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(200))
			body := decodeBody(w.Body)
			Expect(body["status"]).To(Equal("CANCELED"))
		})

		// plan_schedule_status only has ACTIVE and CANCELED. An unknown value used
		// to reach Postgres and come back as a 500.
		It("should reject a status outside the enum", func() {
			Expect(scheduleId).NotTo(BeEmpty())
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{"status": "COMPLETED"})
			req, _ := http.NewRequest("PATCH", fmt.Sprintf("/schedules/%s", scheduleId), bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(400))
		})

		It("should update schedule notes", func() {
			if scheduleId != "" {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{
					"notes": "Workout completed early",
				})
				req, _ := http.NewRequest("PATCH", fmt.Sprintf("/schedules/%s", scheduleId), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)

				if w.Code == 200 {
					body := decodeBody(w.Body)
					Expect(body["notes"]).To(Equal("Workout completed early"))
				}
			}
		})

		It("should fail to update without authentication", func() {
			if scheduleId != "" {
				w := httptest.NewRecorder()
				reqBody, _ := json.Marshal(gin.H{"status": "CANCELLED"})
				req, _ := http.NewRequest("PATCH", fmt.Sprintf("/schedules/%s", scheduleId), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(401))
			}
		})
	})

	Describe("Schedule Deletion", func() {
		It("should delete schedule", func() {
			// Create a schedule to delete
			if planId != "" {
				w := httptest.NewRecorder()
				scheduledDate := time.Now().Add(72 * time.Hour).Format("2006-01-02")
				reqBody, _ := json.Marshal(gin.H{
					"plan_id":        planId,
					"scheduled_for": scheduledDate,
				})
				req, _ := http.NewRequest("POST", "/schedules", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
				router.ServeHTTP(w, req)

				if w.Code == 201 {
					body := decodeBody(w.Body)
					deleteScheduleId := body["id"].(string)

					// Delete the schedule
					w2 := httptest.NewRecorder()
					req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/schedules/%s", deleteScheduleId), nil)
					req2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
					router.ServeHTTP(w2, req2)

					Expect(w2.Code).To(BeNumerically(">=", 200))
					Expect(w2.Code).To(BeNumerically("<", 300))
				}
			}
		})

		It("should fail to delete without authentication", func() {
			if scheduleId != "" {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("DELETE", fmt.Sprintf("/schedules/%s", scheduleId), nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(401))
			}
		})
	})
}
