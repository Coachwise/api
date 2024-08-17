package tests_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ping Route", func() {
	var (
		router *gin.Engine
		db     *sqlx.DB
	)

	BeforeEach(func() {
		db, router = setupTestEnvironment()
	})

	AfterEach(func() {
		teardownTestEnvironment(db)
	})

	Context("GET /ping", func() {
		It("should return status 200 and message pong", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/ping", nil)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(200))
			bodyExpect(decodeBody(w.Body), gin.H{"message": "pong"})
		})
	})
})
