package tests_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"

	"coachwise/src/config"

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

	It("uploads a file and stores it under its kind and month", func() {
		resp := upload(token, "shot.png", pngBytes)
		Expect(resp.Code).To(Equal(http.StatusOK), resp.Body.String())

		var media gin.H
		json.NewDecoder(resp.Body).Decode(&media)

		// The URL must be absolute — the app stores it verbatim — and keyed by
		// kind/year/month so no single directory grows without bound.
		url, _ := media["url"].(string)
		Expect(url).To(MatchRegexp(`/uploads/media/\d{4}/\d{2}/[0-9a-f-]{36}\.png$`))

		key := url[strings.Index(url, "/uploads/")+len("/uploads/"):]
		Expect(filepath.Join(config.Config.Storage.Dir, key)).To(BeAnExistingFile())
	})

	It("rejects a file whose bytes are not an allowed type", func() {
		// Named .png, but it's a text file. The sniffed type is what counts.
		resp := upload(token, "evil.png", []byte("#!/bin/sh\necho pwned\n"))
		Expect(resp.Code).To(Equal(http.StatusBadRequest), resp.Body.String())
	})
}

// A one-pixel PNG: enough for http.DetectContentType to call it image/png.
var pngBytes = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89,
}

func upload(token, filename string, content []byte) *httptest.ResponseRecorder {
	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", filename)
	Expect(err).NotTo(HaveOccurred())
	part.Write(content)
	Expect(w.Close()).To(Succeed())

	req := httptest.NewRequest(http.MethodPost, "/media/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
