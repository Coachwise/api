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

// createExercise creates an exercise owned by the token holder and returns its id.
func createExercise(token, name string) string {
	w := httptest.NewRecorder()
	body, _ := json.Marshal(gin.H{
		"name": name, "description": "desc", "public": true, "sport_type": "STRENGTH",
		"sets": []gin.H{{"name": "Set 1", "rest_time": 30e9, "rep_count": 10}},
	})
	req, _ := http.NewRequest("POST", "/exercises", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(201))
	return decodeBody(w.Body)["id"].(string)
}

// selfAssessmentCount returns how many self-assessment requests an athlete has.
func selfAssessmentCount(athleteID string) int {
	var n int
	err := db.QueryRow(
		"SELECT count(*) FROM test_requests WHERE test_id IS NULL AND athlete_id = $1",
		athleteID,
	).Scan(&n)
	Expect(err).NotTo(HaveOccurred())
	return n
}

func assessmentTestsGroup() {
	var (
		coachToken, coachID     string
		athleteToken, athleteID string
		otherToken              string
		exID, exID2, exID3      string
		testID, itemID          string
		requestID               string
		protoID, protoItemID    string
	)

	Describe("Setup", func() {
		It("creates a coach, an athlete, a non-coach, and exercises", func() {
			coachToken, coachID = registerVerifiedUser("atcoach@test.com", "atcoach")
			athleteToken, athleteID = registerVerifiedUser("atathlete@test.com", "atathlete")
			otherToken, _ = registerVerifiedUser("atother@test.com", "atother")
			makeCoach(coachID)
			exID = createExercise(coachToken, "Bench Press")
			exID2 = createExercise(coachToken, "Pull Up")
			exID3 = createExercise(athleteToken, "20mm Hang")
			Expect(coachID).NotTo(BeEmpty())
			Expect(exID).NotTo(BeEmpty())
		})
	})

	Describe("Tests CRUD", func() {
		It("lets any user create a test (personal protocol)", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"name": "Personal protocol"})
			req, _ := http.NewRequest("POST", "/tests", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
		})

		It("creates a test with items", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name":  "Strength Assessment",
				"items": []gin.H{{"exercise_id": exID, "track_weight": true}},
			})
			req, _ := http.NewRequest("POST", "/tests", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			testID = b["id"].(string)
			Expect(b["item_count"]).To(BeEquivalentTo(1))
			items, _ := b["items"].([]interface{})
			Expect(items).To(HaveLen(1))
			itemID = items[0].(map[string]interface{})["id"].(string)
		})

		It("lists the coach's tests", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["total"]).To(BeEquivalentTo(1))
		})
	})

	Describe("Request → submit → seen", func() {
		It("coach requests the athlete to take the test", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"athlete_id": athleteID, "note": "please complete"})
			req, _ := http.NewRequest("POST", "/tests/"+testID+"/request", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			requestID = b["id"].(string)
			Expect(b["status"]).To(Equal("PENDING"))
		})

		It("athlete sees the assigned request", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/requests/assigned", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["total"]).To(BeNumerically(">=", 1))
		})

		It("athlete submits records → SUBMITTED", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"records": []gin.H{{"test_item_id": itemID, "weight": 100}}})
			req, _ := http.NewRequest("POST", "/tests/requests/"+requestID+"/submit", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["status"]).To(Equal("SUBMITTED"))
		})

		It("a PR appears as soon as it's submitted (no approval gate)", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			records, _ := decodeBody(w.Body)["records"].([]interface{})
			Expect(records).To(HaveLen(1))
			Expect(records[0].(map[string]interface{})["best_weight"]).To(BeEquivalentTo(100))
		})

		It("only a coach can mark seen", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/tests/requests/"+requestID+"/seen", nil)
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(403)) // not a coach
		})

		It("coach marks the submission seen", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/tests/requests/"+requestID+"/seen", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["status"]).To(Equal("SEEN"))
		})

		It("the PR persists after the coach has seen it", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			records, _ := decodeBody(w.Body)["records"].([]interface{})
			Expect(records).To(HaveLen(1))
		})
	})

	Describe("Athlete self-assessment", func() {
		It("connects the coach and athlete", func() {
			_, err := db.Exec(
				"INSERT INTO connections (user1_id, user2_id) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) ON CONFLICT DO NOTHING",
				coachID, athleteID,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("athlete records a self-assessment (no coach, no template)", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name":    "Morning check",
				"records": []gin.H{{"exercise_id": exID2, "reps": 12, "weight": 20}},
			})
			req, _ := http.NewRequest("POST", "/tests/requests/self", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			Expect(b["status"]).To(Equal("SUBMITTED"))
			Expect(b["test"].(map[string]interface{})["self"]).To(BeTrue())
			Expect(b["test"].(map[string]interface{})["name"]).To(Equal("Morning check"))
		})

		It("rejects a self-assessment with no usable results (no orphan row)", func() {
			before := selfAssessmentCount(athleteID)
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name":    "empty",
				"records": []gin.H{{"exercise_id": exID2}}, // no reps/weight/time
			})
			req, _ := http.NewRequest("POST", "/tests/requests/self", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
			Expect(selfAssessmentCount(athleteID)).To(Equal(before)) // rolled back, no orphan
		})

		It("a self-assessment PR shows on the athlete's profile", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			records, _ := decodeBody(w.Body)["records"].([]interface{})
			Expect(records).To(HaveLen(2)) // bench (coach) + pull up (self)
		})

		It("the connected coach sees the self-assessment as submitted", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/requests?status=SUBMITTED", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			Expect(items).To(HaveLen(1))
			self := items[0].(map[string]interface{})["test"].(map[string]interface{})["self"]
			Expect(self).To(BeTrue())
		})

		It("the coach can mark a connected athlete's self-assessment seen", func() {
			// find the self-assessment id from the coach's submitted list
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/requests?status=SUBMITTED", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			selfID := items[0].(map[string]interface{})["id"].(string)

			w = httptest.NewRecorder()
			req, _ = http.NewRequest("POST", "/tests/requests/"+selfID+"/seen", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["status"]).To(Equal("SEEN"))
		})
	})

	Describe("Achievements (coach-granted badges)", func() {
		It("coach grants a badge", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"athlete_id": athleteID, "title": "12-Week Program"})
			req, _ := http.NewRequest("POST", "/achievements", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
		})

		It("the badge appears on the athlete's achievements", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			badges, _ := decodeBody(w.Body)["badges"].([]interface{})
			Expect(badges).To(HaveLen(1))
		})
	})

	Describe("Personal protocols (repeatable self-assessments)", func() {
		It("an athlete creates their own protocol", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name":  "Climbing assessment",
				"items": []gin.H{{"exercise_id": exID3, "track_weight": true, "track_time": true}},
			})
			req, _ := http.NewRequest("POST", "/tests", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			protoID = b["id"].(string)
			items, _ := b["items"].([]interface{})
			protoItemID = items[0].(map[string]interface{})["id"].(string)
		})

		It("records two dated runs of the protocol", func() {
			for _, weight := range []int{50, 60} {
				w := httptest.NewRecorder()
				body, _ := json.Marshal(gin.H{"records": []gin.H{{"test_item_id": protoItemID, "weight": weight, "time": 7}}})
				req, _ := http.NewRequest("POST", "/tests/"+protoID+"/run", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+athleteToken)
				router.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(201))
				Expect(decodeBody(w.Body)["status"]).To(Equal("SUBMITTED"))
			}
		})

		It("keeps every run in the history", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/"+protoID+"/runs", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["total"]).To(BeEquivalentTo(2))
		})

		It("the PR reflects the best run", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			records, _ := decodeBody(w.Body)["records"].([]interface{})
			var best interface{}
			for _, r := range records {
				m := r.(map[string]interface{})
				if m["exercise_id"] == exID3 {
					best = m["best_weight"]
				}
			}
			Expect(best).To(BeEquivalentTo(60))
		})

		It("only the owner can run their protocol", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"records": []gin.H{{"test_item_id": protoItemID, "weight": 99}}})
			req, _ := http.NewRequest("POST", "/tests/"+protoID+"/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})
	})

	Describe("Coach-assigned protocols (run like personal ones)", func() {
		It("the assigned test shows in the athlete's assigned protocols", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/assigned", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			found := false
			for _, it := range items {
				if it.(map[string]interface{})["id"] == testID {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})

		It("the athlete can run a coach-assigned protocol repeatedly", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"records": []gin.H{{"test_item_id": itemID, "weight": 110}}})
			req, _ := http.NewRequest("POST", "/tests/"+testID+"/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
		})

		It("the assigned protocol's run history is visible to the athlete", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/"+testID+"/runs", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			// The original coach submission + the new run.
			Expect(decodeBody(w.Body)["total"]).To(BeNumerically(">=", 2))
		})

		It("a non-assigned user cannot run the coach's protocol", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"records": []gin.H{{"test_item_id": itemID, "weight": 1}}})
			req, _ := http.NewRequest("POST", "/tests/"+testID+"/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})

		It("the coach sees the assignment with the client + run stats", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/assignments", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			var match map[string]interface{}
			for _, it := range items {
				m := it.(map[string]interface{})
				if m["test_id"] == testID && m["athlete_id"] == athleteID {
					match = m
				}
			}
			Expect(match).NotTo(BeNil())
			Expect(match["runs_count"]).To(BeNumerically(">=", 2))
			Expect(match["athlete"].(map[string]interface{})["id"]).To(Equal(athleteID))
		})

		It("the coach can view a client's run history for the protocol", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/"+testID+"/runs?athlete="+athleteID, nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["total"]).To(BeNumerically(">=", 2))
		})

		It("a coach who doesn't own the protocol cannot view its runs", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/tests/"+testID+"/runs?athlete="+athleteID, nil)
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})
	})

	Describe("Profile trophy case (layout + stats)", func() {
		It("exposes an active-clients count on the profile", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+coachID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			_, ok := decodeBody(w.Body)["active_clients"]
			Expect(ok).To(BeTrue())
		})

		It("the athlete hides a record from their layout", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"order": []string{}, "hidden": []string{"record:" + exID2}})
			req, _ := http.NewRequest("PUT", "/achievements/layout", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
		})

		It("hides that record from other viewers but keeps it for the owner", func() {
			// Other viewer (coach): the hidden pull-up record is gone.
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			records, _ := decodeBody(w.Body)["records"].([]interface{})
			for _, r := range records {
				Expect(r.(map[string]interface{})["exercise_id"]).NotTo(Equal(exID2))
			}

			// Owner still sees it, and the hidden key is in their layout.
			w = httptest.NewRecorder()
			req, _ = http.NewRequest("GET", "/users/"+athleteID+"/achievements", nil)
			req.Header.Set("Authorization", "Bearer "+athleteToken)
			router.ServeHTTP(w, req)
			body := decodeBody(w.Body)
			ownerRecords, _ := body["records"].([]interface{})
			found := false
			for _, r := range ownerRecords {
				if r.(map[string]interface{})["exercise_id"] == exID2 {
					found = true
				}
			}
			Expect(found).To(BeTrue())
			layout := body["layout"].(map[string]interface{})
			hidden, _ := layout["hidden"].([]interface{})
			Expect(hidden).To(ContainElement("record:" + exID2))
		})
	})
}
