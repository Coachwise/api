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

var _ = Describe("Auth Group", func() {
	var (
		router *gin.Engine
		// db     *sqlx.DB
	)

	BeforeEach(func() {
		_, router = setupTestEnvironment()
	})

	Context("POST /auth/register", func() {
		It("should return status 200 with jwt tokens", func() {
			register(router)
		})
	})

	AfterEach(func() {
		// teardownTestEnvironment(db)
	})
})

func register(r *gin.Engine) {
	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(users_data[0])
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	body := decodeBody(w.Body)

	Expect(w.Code).To(Equal(200))
	bodyExpect(body, gin.H{"access_token": "<ANY>"})
}
