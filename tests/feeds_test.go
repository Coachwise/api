package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"coachwise/src/app/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var feedGroup = func() {
	var (
		token  string
		feedID string
	)

	BeforeEach(func() {
		if token == "" {
			// Register user
			registerPayload := gin.H{
				"first_name": "Feed",
				"last_name":  "Tester",
				"username":   "feedtester",
				"email":      "feedtester@test.com",
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

			// Activate user without OTP
			db.Exec("UPDATE users SET status = 'ACTIVE' WHERE id = $1", userID)

			// Login
			loginPayload := gin.H{
				"username": "feedtester@test.com",
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
		}

		// Create a fresh feed for each test
		payload := gin.H{
			"body": "Bootstrap feed",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/feeds", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())

		var result gin.H
		json.NewDecoder(resp.Body).Decode(&result)
		feedID = result["id"].(string)

		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM feeds WHERE id=$1", feedID)
		Expect(err).To(BeNil())
		Expect(count).To(Equal(1))
	})

	Describe("Feeds CRUD", func() {
		It("creates a feed", func() {
			payload := gin.H{
				"body": "Great session today",
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/feeds", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())

			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)
			Expect(result["id"]).NotTo(BeNil())
			Expect(result["visibility"]).To(Equal("PUBLIC"))

			feedID = result["id"].(string)
		})

		It("fetches a feed by id", func() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/feeds/%s", feedID), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())
			var result gin.H
			json.NewDecoder(resp.Body).Decode(&result)
			Expect(result["id"]).To(Equal(feedID))
			Expect(result["like_count"]).To(BeNumerically(">=", 0))
			Expect(result["comment_count"]).To(BeNumerically(">=", 0))
		})

		It("lists feeds and includes created feed", func() {
			req := httptest.NewRequest(http.MethodGet, "/feeds", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())
			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body)

			feeds := body["feeds"].([]interface{})
			Expect(len(feeds)).To(BeNumerically(">=", 1))

			found := false
			for _, f := range feeds {
				entry := f.(map[string]interface{})
				if entry["id"] == feedID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("likes and unlikes a feed", func() {
			likeReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/feeds/%s/like", feedID), nil)
			likeReq.Header.Set("Authorization", "Bearer "+token)
			likeResp := httptest.NewRecorder()
			router.ServeHTTP(likeResp, likeReq)
			Expect(likeResp.Code).To(Equal(http.StatusOK), likeResp.Body.String())

			var likeCount int
			err := db.Get(&likeCount, "SELECT COUNT(*) FROM feed_likes WHERE feed_id=$1", feedID)
			Expect(err).To(BeNil())
			Expect(likeCount).To(Equal(1))

			var userID string
			Expect(db.Get(&userID, "SELECT id FROM users WHERE email=$1", "feedtester@test.com")).To(Succeed())
			likedState, likeErr := models.HasUserLikedFeed(context.Background(), uuid.MustParse(feedID), uuid.MustParse(userID))
			Expect(likeErr).To(BeNil())
			Expect(likedState).To(BeTrue())

			getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/feeds/%s", feedID), nil)
			getReq.Header.Set("Authorization", "Bearer "+token)
			getResp := httptest.NewRecorder()
			router.ServeHTTP(getResp, getReq)
			var feed gin.H
			json.NewDecoder(getResp.Body).Decode(&feed)
			Expect(feed["liked"]).To(Equal(true))
			Expect(feed["like_count"]).To(Equal(float64(1)))

			unlikeReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/feeds/%s/like", feedID), nil)
			unlikeReq.Header.Set("Authorization", "Bearer "+token)
			unlikeResp := httptest.NewRecorder()
			router.ServeHTTP(unlikeResp, unlikeReq)
			Expect(unlikeResp.Code).To(Equal(http.StatusOK), unlikeResp.Body.String())
		})

		It("adds and lists comments", func() {
			commentPayload := gin.H{"body": "Nice work!"}
			body, _ := json.Marshal(commentPayload)

			commentReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/feeds/%s/comments", feedID), bytes.NewBuffer(body))
			commentReq.Header.Set("Content-Type", "application/json")
			commentReq.Header.Set("Authorization", "Bearer "+token)
			commentResp := httptest.NewRecorder()
			router.ServeHTTP(commentResp, commentReq)
			Expect(commentResp.Code).To(Equal(http.StatusOK), commentResp.Body.String())

			listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/feeds/%s/comments", feedID), nil)
			listReq.Header.Set("Authorization", "Bearer "+token)
			listResp := httptest.NewRecorder()
			router.ServeHTTP(listResp, listReq)

			Expect(listResp.Code).To(Equal(http.StatusOK), listResp.Body.String())
			var bodyResp map[string]interface{}
			json.NewDecoder(listResp.Body).Decode(&bodyResp)
			comments := bodyResp["comments"].([]interface{})
			Expect(len(comments)).To(BeNumerically(">=", 1))
		})
	})
}
