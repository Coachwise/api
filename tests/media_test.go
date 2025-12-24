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

var mediaGroup = func() {
	var token string

	BeforeEach(func() {
		if token != "" {
			return
		}

		registerPayload := gin.H{
			"first_name": "Media",
			"last_name":  "Tester",
			"username":   "mediatestr",
			"email":      "mediatestr@test.com",
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
		db.Exec("UPDATE users SET status = 'ACTIVE' WHERE id = $1", userID)

		loginPayload := gin.H{
			"username": "mediatestr@test.com",
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
	})

	It("creates media and lists it", func() {
		createPayload := gin.H{
			"url":        "http://example.com/demo.jpg",
			"filename":   "demo.jpg",
			"size_bytes": 12345,
		}
		body, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/media", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())

		var media gin.H
		json.NewDecoder(resp.Body).Decode(&media)
		Expect(media["id"]).NotTo(BeNil())

		listReq := httptest.NewRequest(http.MethodGet, "/media", nil)
		listReq.Header.Set("Authorization", "Bearer "+token)
		listResp := httptest.NewRecorder()
		router.ServeHTTP(listResp, listReq)
		Expect(listResp.Code).To(Equal(http.StatusOK), listResp.Body.String())

		var list []map[string]interface{}
		json.NewDecoder(listResp.Body).Decode(&list)
		Expect(len(list)).To(BeNumerically(">=", 1))
	})
}
