package tests_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func pingGroup() {
	It("should return status 200 and message pong", func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/ping", nil)
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(200))
		bodyExpect(decodeBody(w.Body), gin.H{"message": "pong"})
	})
}
