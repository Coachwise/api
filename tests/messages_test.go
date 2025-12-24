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

var messagesGroup = func() {
	var (
		tokenA    string
		tokenB    string
		userAID   string
		userBID   string
		mediaID   string
		messageID string
	)

	BeforeEach(func() {
		if tokenA != "" && tokenB != "" {
			return
		}

		// User A
		registerPayloadA := gin.H{
			"first_name": "MsgA",
			"last_name":  "Tester",
			"username":   "msga",
			"email":      "msga@test.com",
			"password":   "Password123!",
		}
		registerBodyA, _ := json.Marshal(registerPayloadA)
		regReqA := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(registerBodyA))
		regReqA.Header.Set("Content-Type", "application/json")
		regRespA := httptest.NewRecorder()
		router.ServeHTTP(regRespA, regReqA)
		Expect(regRespA.Code).To(Equal(http.StatusOK))
		var regResultA gin.H
		json.NewDecoder(regRespA.Body).Decode(&regResultA)
		userAID = regResultA["id"].(string)
		db.Exec("UPDATE users SET status = 'ACTIVE' WHERE id = $1", userAID)

		loginPayloadA := gin.H{"username": "msga@test.com", "password": "Password123!"}
		loginBodyA, _ := json.Marshal(loginPayloadA)
		loginReqA := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(loginBodyA))
		loginReqA.Header.Set("Content-Type", "application/json")
		loginRespA := httptest.NewRecorder()
		router.ServeHTTP(loginRespA, loginReqA)
		Expect(loginRespA.Code).To(Equal(http.StatusOK))
		var loginResultA gin.H
		json.NewDecoder(loginRespA.Body).Decode(&loginResultA)
		tokenA = loginResultA["token"].(string)

		// User B
		registerPayloadB := gin.H{
			"first_name": "MsgB",
			"last_name":  "Tester",
			"username":   "msgb",
			"email":      "msgb@test.com",
			"password":   "Password123!",
		}
		registerBodyB, _ := json.Marshal(registerPayloadB)
		regReqB := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(registerBodyB))
		regReqB.Header.Set("Content-Type", "application/json")
		regRespB := httptest.NewRecorder()
		router.ServeHTTP(regRespB, regReqB)
		Expect(regRespB.Code).To(Equal(http.StatusOK))
		var regResultB gin.H
		json.NewDecoder(regRespB.Body).Decode(&regResultB)
		userBID = regResultB["id"].(string)
		db.Exec("UPDATE users SET status = 'ACTIVE' WHERE id = $1", userBID)

		loginPayloadB := gin.H{"username": "msgb@test.com", "password": "Password123!"}
		loginBodyB, _ := json.Marshal(loginPayloadB)
		loginReqB := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(loginBodyB))
		loginReqB.Header.Set("Content-Type", "application/json")
		loginRespB := httptest.NewRecorder()
		router.ServeHTTP(loginRespB, loginReqB)
		Expect(loginRespB.Code).To(Equal(http.StatusOK))
		var loginResultB gin.H
		json.NewDecoder(loginRespB.Body).Decode(&loginResultB)
		tokenB = loginResultB["token"].(string)

		// Create media for attachment (by user A)
		mediaPayload := gin.H{
			"url":      "http://example.com/msg.jpg",
			"filename": "msg.jpg",
		}
		mediaBody, _ := json.Marshal(mediaPayload)
		mediaReq := httptest.NewRequest(http.MethodPost, "/media", bytes.NewBuffer(mediaBody))
		mediaReq.Header.Set("Content-Type", "application/json")
		mediaReq.Header.Set("Authorization", "Bearer "+tokenA)
		mediaResp := httptest.NewRecorder()
		router.ServeHTTP(mediaResp, mediaReq)
		Expect(mediaResp.Code).To(Equal(http.StatusOK), mediaResp.Body.String())
		var mediaResult gin.H
		json.NewDecoder(mediaResp.Body).Decode(&mediaResult)
		mediaID = mediaResult["id"].(string)
	})

	It("sends, lists, and marks messages as read", func() {
		sendPayload := gin.H{
			"recipient_id": userBID,
			"body":         "Great progress this week!",
			"media_id":     mediaID,
		}
		body, _ := json.Marshal(sendPayload)
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenA)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())

		var msg gin.H
		json.NewDecoder(resp.Body).Decode(&msg)
		messageID = msg["id"].(string)
		Expect(msg["media_id"]).To(Equal(mediaID))

		// List messages (user A view)
		listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/messages/%s?limit=10", userBID), nil)
		listReq.Header.Set("Authorization", "Bearer "+tokenA)
		listResp := httptest.NewRecorder()
		router.ServeHTTP(listResp, listReq)
		Expect(listResp.Code).To(Equal(http.StatusOK), listResp.Body.String())
		var msgs []map[string]interface{}
		json.NewDecoder(listResp.Body).Decode(&msgs)
		Expect(len(msgs)).To(BeNumerically(">=", 1))

		// Threads list
		threadsReq := httptest.NewRequest(http.MethodGet, "/messages/threads", nil)
		threadsReq.Header.Set("Authorization", "Bearer "+tokenA)
		threadsResp := httptest.NewRecorder()
		router.ServeHTTP(threadsResp, threadsReq)
		Expect(threadsResp.Code).To(Equal(http.StatusOK), threadsResp.Body.String())

		// Mark read as recipient (user B)
		markReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/messages/%s/read", userAID), nil)
		markReq.Header.Set("Authorization", "Bearer "+tokenB)
		markResp := httptest.NewRecorder()
		router.ServeHTTP(markResp, markReq)
		Expect(markResp.Code).To(Equal(http.StatusOK), markResp.Body.String())

		var readAt *string
		err := db.Get(&readAt, "SELECT read_at::text FROM messages WHERE id=$1", messageID)
		Expect(err).To(BeNil())
		Expect(readAt).NotTo(BeNil())
	})
}
