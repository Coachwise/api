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

// This is the shape of the outage that took phone login down in production: the
// database circuit breaker counts a query as FAILED when it returns "no rows".
// A handful of people typing an unknown email — or signing up with a phone that
// naturally isn't in the users table yet — opens the breaker, and every request
// after that gets "circuit breaker is open" until someone restarts the API. It
// never heals on its own, because the half-open probe is the same not-found
// lookup that failed in the first place.
//
// The breaker is meant to shed load when the DATABASE is unwell. "No rows" is
// the database working perfectly.
func resilienceGroup() {
	It("stays up after a run of failed logins", func() {
		for i := 0; i < 6; i++ {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"username": fmt.Sprintf("nobody%d@nowhere.test", i),
				"password": "Password123!",
			})
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			Expect(w.Code).To(BeNumerically(">=", 400))
			Expect(w.Code).To(BeNumerically("<", 500), "an unknown user is a client error, never a 500")
		}

		// The database is fine. A real user must still be able to log in.
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authTokens[0]))
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(200), "failed logins must not take the API down")
	})
}
