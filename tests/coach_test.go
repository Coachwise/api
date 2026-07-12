package tests_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func decisionToken(appID string) string {
	var token string
	Expect(db.Get(&token, "SELECT decision_token FROM coach_applications WHERE id = $1", appID)).To(Succeed())
	return token
}

func isCoach(token string) bool {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	body := decodeBody(w.Body)
	coach, _ := body["is_coach"].(bool)
	return coach
}

func coachApplicationsGroup() {
	var (
		tokenA string
		appA   string
	)

	Describe("Applying", func() {
		It("requires authentication", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"full_name": "X", "specialty": "CrossFit", "certifications": "NASM"})
			req, _ := http.NewRequest("POST", "/coaches/apply", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("submits an application (PENDING, token not exposed)", func() {
			tokenA, _ = registerVerifiedUser("coachapp@test.com", "coachapp")
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"full_name":        "Coach Apphdr",
				"specialty":        "Rock Climbing",
				"experience_years": 6,
				"certifications":   "USAC L2",
			})
			req, _ := http.NewRequest("POST", "/coaches/apply", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200), w.Body.String())
			resp := decodeBody(w.Body)
			Expect(resp["status"]).To(Equal("PENDING"))
			Expect(resp["decision_token"]).To(BeNil())
			appA = resp["id"].(string)
			Expect(appA).NotTo(BeEmpty())
		})

		It("validates required fields", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"full_name": "No Specialty"})
			req, _ := http.NewRequest("POST", "/coaches/apply", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("returns the user's latest application", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/coaches/application", nil)
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			resp := decodeBody(w.Body)
			Expect(resp["id"]).To(Equal(appA))
			Expect(resp["status"]).To(Equal("PENDING"))
		})
	})

	Describe("Deciding via capability links", func() {
		It("rejects an invalid decision token", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/coaches/applications/%s/approve?token=wrong", appA), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
			Expect(isCoach(tokenA)).To(BeFalse())
		})

		It("approves via the link and makes the user a coach", func() {
			token := decisionToken(appA)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/coaches/applications/%s/approve?token=%s", appA, token), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200), w.Body.String())
			Expect(isCoach(tokenA)).To(BeTrue())
		})

		It("does not re-decide an already-decided application", func() {
			token := decisionToken(appA)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/coaches/applications/%s/reject?token=%s", appA, token), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Rejecting", func() {
		It("rejects an application and leaves the user a non-coach", func() {
			tokenB, _ := registerVerifiedUser("coachapp2@test.com", "coachapp2")
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"full_name": "Coach Bee", "specialty": "CrossFit", "certifications": "CF L1"})
			req, _ := http.NewRequest("POST", "/coaches/apply", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokenB)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200), w.Body.String())
			appB := decodeBody(w.Body)["id"].(string)

			token := decisionToken(appB)
			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("GET", fmt.Sprintf("/coaches/applications/%s/reject?token=%s", appB, token), nil)
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(200), w2.Body.String())

			Expect(isCoach(tokenB)).To(BeFalse())

			w3 := httptest.NewRecorder()
			req3, _ := http.NewRequest("GET", "/coaches/application", nil)
			req3.Header.Set("Authorization", "Bearer "+tokenB)
			router.ServeHTTP(w3, req3)
			Expect(decodeBody(w3.Body)["status"]).To(Equal("REJECTED"))
		})
	})
}
