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

	// Anyone can build an exercise now, but later groups (plans, packages,
	// assessments) still expect the shared user to be a coach — this promotion
	// runs first, so keep it.
	BeforeEach(func() {
		db.MustExec("UPDATE users SET is_coach = true WHERE email = $1", usersData[0]["email"])
	})

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

		It("should force created exercises to be personal, ignoring public:true", func() {
			// The API refuses to publish to the shared library; public is server-set
			// to false regardless of what the client sends.
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Public Exercise",
				"description": "This is a public exercise",
				"public":      true,
				"sport_type":  "GENERAL",
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
			Expect(body["public"]).To(Equal(false))
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
				"sport_type":  "MOBILITY",
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

		It("should create climbing exercise", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Hangboard Max Hangs",
				"description": "Fingerboard training",
				"public":      true,
				"sport_type":  "CLIMBING",
				"sets": []gin.H{
					{"name": "Warmup", "rest_time": 180e9, "duration": 7e9},
					{"name": "Max Hang", "rest_time": 180e9, "duration": 10e9},
					{"name": "Max Hang", "rest_time": 180e9, "duration": 10e9},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(201))
			Expect(body["sport_type"]).To(Equal("CLIMBING"))
			sets := body["sets"].([]interface{})
			Expect(len(sets)).To(Equal(3))
		})

		It("should create cardio exercise", func() {
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"name":        "Running Intervals",
				"description": "Cardio endurance training",
				"public":      true,
				"sport_type":  "CARDIO",
				"sets": []gin.H{
					{"name": "Sprint", "rest_time": 60e9, "duration": 30e9},
					{"name": "Sprint", "rest_time": 60e9, "duration": 30e9},
					{"name": "Sprint", "rest_time": 60e9, "duration": 30e9},
				},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)

			body := decodeBody(w.Body)
			Expect(w.Code).To(Equal(201))
			Expect(body["sport_type"]).To(Equal("CARDIO"))
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
			Expect(w.Code).To(Equal(404))
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

			Expect(w.Code).To(Equal(200))
			body := decodeBody(w.Body)
			Expect(body["total"]).To(BeNumerically(">=", 2))
			Expect(len(body["items"].([]interface{}))).To(BeNumerically(">=", 2))
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

			Expect(w.Code).To(Equal(200))
			body := decodeBody(w.Body)
			Expect(len(body["items"].([]interface{}))).To(BeNumerically(">=", 1))
		})
	})

	Describe("Exercise Update", func() {
		It("should update exercise details", func() {
			updatedData := gin.H{
				"name":        "updated",
				"description": "updated",
				"public":      false,
				"sport_type":  "STRENGTH",
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
				"sport_type":  "STRENGTH",
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
			Expect(w.Code).To(Equal(404))
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
			Expect(w2.Code).To(Equal(404))
		})

		It("should fail to delete non-existent exercise", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/exercises/00000000-0000-0000-0000-000000000000", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})

		It("should fail to delete exercise without authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/exercises/%s", publicExerciseId), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})

	Describe("Exercise Visibility", func() {
		var bToken, userAID, userBID string

		// Register a second user (B) once, so specs can assert what B can and
		// cannot see of user A's exercises.
		BeforeEach(func() {
			db.Get(&userAID, "SELECT id FROM users WHERE email = $1", usersData[0]["email"])
			if bToken != "" {
				return
			}
			w := httptest.NewRecorder()
			reqBody, _ := json.Marshal(gin.H{
				"first_name": "Visibility",
				"last_name":  "Bee",
				"username":   "visibilitybee",
				"email":      "visibility_b@test.com",
				"password":   "password123",
			})
			req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			otp := struct{ Code string }{}
			db.Get(&otp, "SELECT code FROM otps WHERE email = 'visibility_b@test.com' LIMIT 1")
			w2 := httptest.NewRecorder()
			reqBody2, _ := json.Marshal(gin.H{"email": "visibility_b@test.com", "code": otp.Code})
			req2, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(reqBody2))
			req2.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w2, req2)
			body := decodeBody(w2.Body)
			bToken = body["access_token"].(string)
			db.Get(&userBID, "SELECT id FROM users WHERE email = 'visibility_b@test.com'")
		})

		It("should hide user A's personal exercise from user B (detail 404)", func() {
			var exID string
			db.Get(&exID, `INSERT INTO exercises (user_id, name, description, public, sport_type)
				VALUES ($1, 'A private', 'x', false, 'GENERAL') RETURNING id`, userAID)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", exID), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})

		It("should keep user A's personal exercise out of user B's list", func() {
			var exID string
			db.Get(&exID, `INSERT INTO exercises (user_id, name, description, public, sport_type)
				VALUES ($1, 'A private listed', 'x', false, 'GENERAL') RETURNING id`, userAID)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/exercises?limit=100", nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			body := decodeBody(w.Body)
			for _, item := range body["items"].([]interface{}) {
				Expect(item.(map[string]interface{})["id"]).ToNot(Equal(exID))
			}
		})

		It("should let user B open user A's personal exercise via an assigned plan", func() {
			var exID, planID string
			db.Get(&exID, `INSERT INTO exercises (user_id, name, description, public, sport_type)
				VALUES ($1, 'A plan exercise', 'x', false, 'GENERAL') RETURNING id`, userAID)
			db.Get(&planID, `INSERT INTO plans (user_id, name, public)
				VALUES ($1, 'A plan', false) RETURNING id`, userAID)
			db.MustExec(`INSERT INTO plan_exercises (exercise_id, plan_id, exercise_order, rest_time)
				VALUES ($1, $2, 1, 0)`, exID, planID)
			db.MustExec(`INSERT INTO plan_assignees (plan_id, user_id, assigner)
				VALUES ($1, $2, $3)`, planID, userBID, userAID)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", exID), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
		})

		It("should let anyone view a seeded library exercise but nobody edit or delete it", func() {
			var libID string
			db.Get(&libID, `INSERT INTO exercises (user_id, name, description, public, sport_type)
				VALUES (NULL, 'Library exercise', 'x', true, 'GENERAL') RETURNING id`)

			// Visible to user B (it is public).
			wGet := httptest.NewRecorder()
			reqGet, _ := http.NewRequest("GET", fmt.Sprintf("/exercises/%s", libID), nil)
			reqGet.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bToken))
			router.ServeHTTP(wGet, reqGet)
			Expect(wGet.Code).To(Equal(200))

			// But a user_id-NULL row is owned by nobody, so edits are forbidden.
			wPut := httptest.NewRecorder()
			putBody, _ := json.Marshal(gin.H{"name": "hijacked", "description": "x"})
			reqPut, _ := http.NewRequest("PUT", fmt.Sprintf("/exercises/%s", libID), bytes.NewBuffer(putBody))
			reqPut.Header.Set("Content-Type", "application/json")
			reqPut.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(wPut, reqPut)
			Expect(wPut.Code).To(Equal(403))

			wDel := httptest.NewRecorder()
			reqDel, _ := http.NewRequest("DELETE", fmt.Sprintf("/exercises/%s", libID), nil)
			reqDel.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
			router.ServeHTTP(wDel, reqDel)
			Expect(wDel.Code).To(Equal(403))
		})
	})

	// A group is an exercise that references other exercises and repeats them as
	// rounds. Children must be SINGLE, which is what keeps groups one level deep.
	Describe("Exercise Groups", func() {
		var childA, childB, groupID string

		newSingle := func(name string) string {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name": name, "description": name, "sport_type": "STRENGTH",
				"sets": []gin.H{{"rest_time": 30e9, "rep_count": 10}},
			})
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+authTokens[0])
			router.ServeHTTP(w, req)
			return decodeBody(w.Body)["id"].(string)
		}

		postGroup := func(payload gin.H) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+authTokens[0])
			router.ServeHTTP(w, req)
			return w
		}

		BeforeEach(func() {
			if childA == "" {
				childA = newSingle("Group child A")
				childB = newSingle("Group child B")
			}
		})

		It("creates a group and returns its items in round order", func() {
			w := postGroup(gin.H{
				"name": "Upper circuit", "description": "circuit", "kind": "GROUP",
				"rounds": 5, "round_rest": 90e9,
				"items": []gin.H{
					{"exercise_id": childA, "rep_count": 15, "rest_time": 30e9},
					{"exercise_id": childB, "rep_count": 10, "rest_time": 0},
				},
			})
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			groupID = b["id"].(string)
			Expect(b["kind"]).To(Equal("GROUP"))
			Expect(b["rounds"]).To(BeEquivalentTo(5))

			items := b["items"].([]interface{})
			Expect(items).To(HaveLen(2))
			first := items[0].(map[string]interface{})
			Expect(first["exercise_id"]).To(Equal(childA))
			Expect(first["item_order"]).To(BeEquivalentTo(1))
			Expect(first["rep_count"]).To(BeEquivalentTo(15))
			// The child exercise rides along so the runner needs no second call.
			Expect(first["exercise"].(map[string]interface{})["name"]).To(Equal("Group child A"))
			Expect(items[1].(map[string]interface{})["item_order"]).To(BeEquivalentTo(2))
		})

		It("returns the group with its items on fetch", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/exercises/"+groupID, nil)
			req.Header.Set("Authorization", "Bearer "+authTokens[0])
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["items"].([]interface{})).To(HaveLen(2))
		})

		It("rejects a group containing another group (one level only)", func() {
			w := postGroup(gin.H{
				"name": "Nested", "description": "nope", "kind": "GROUP", "rounds": 3,
				"items": []gin.H{{"exercise_id": groupID, "rep_count": 5}},
			})
			Expect(w.Code).To(Equal(400))
		})

		It("rejects a group with no exercises", func() {
			w := postGroup(gin.H{
				"name": "Empty", "description": "nope", "kind": "GROUP", "rounds": 3,
				"items": []gin.H{},
			})
			Expect(w.Code).To(Equal(400))
		})

		It("rejects a group with neither rounds nor a duration", func() {
			w := postGroup(gin.H{
				"name": "No repeat", "description": "nope", "kind": "GROUP",
				"items": []gin.H{{"exercise_id": childA, "rep_count": 5}},
			})
			Expect(w.Code).To(Equal(400))
		})

		It("rejects a group with both rounds and a duration", func() {
			w := postGroup(gin.H{
				"name": "Both", "description": "nope", "kind": "GROUP",
				"rounds": 3, "round_duration": 1200e9,
				"items": []gin.H{{"exercise_id": childA, "rep_count": 5}},
			})
			Expect(w.Code).To(Equal(400))
		})

		It("rejects an item with neither reps nor duration", func() {
			w := postGroup(gin.H{
				"name": "Bad item", "description": "nope", "kind": "GROUP", "rounds": 3,
				"items": []gin.H{{"exercise_id": childA}},
			})
			Expect(w.Code).To(Equal(400))
		})

		It("rejects items on a single exercise", func() {
			w := postGroup(gin.H{
				"name": "Not a group", "description": "nope",
				"items": []gin.H{{"exercise_id": childA, "rep_count": 5}},
			})
			Expect(w.Code).To(Equal(400))
		})

		It("accepts a time-capped group", func() {
			w := postGroup(gin.H{
				"name": "AMRAP 20", "description": "amrap", "kind": "GROUP",
				"round_duration": 1200e9,
				"items":          []gin.H{{"exercise_id": childA, "rep_count": 5, "rest_time": 0}},
			})
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			Expect(b["round_duration"]).To(BeEquivalentTo(1200e9))
			Expect(b["rounds"]).To(BeNil())
		})

		It("switching a group back to single clears its rounds and items", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name": "Upper circuit", "description": "now single", "kind": "SINGLE",
				"sets": []gin.H{{"rest_time": 30e9, "rep_count": 10}},
			})
			req, _ := http.NewRequest("PUT", "/exercises/"+groupID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+authTokens[0])
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			b := decodeBody(w.Body)
			Expect(b["kind"]).To(Equal("SINGLE"))
			Expect(b["rounds"]).To(BeNil())
			Expect(b["items"]).To(BeEmpty())
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
				"sport_type":  "STRENGTH",
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
				"sport_type":  "STRENGTH",
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