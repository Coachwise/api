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

// registerVerifiedUser registers a user, verifies its OTP, and returns an access
// token plus the user id. Self-contained so the connection specs don't depend on
// tokens mutated by earlier suites.
func registerVerifiedUser(email, username string) (string, string) {
	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(gin.H{
		"first_name": username,
		"last_name":  "Conn",
		"username":   username,
		"email":      email,
		"password":   "password123",
	})
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(200))

	otp := struct{ Code string }{}
	Expect(db.Get(&otp, "SELECT code FROM otps WHERE email = $1 ORDER BY created_at DESC LIMIT 1", email)).To(Succeed())

	w2 := httptest.NewRecorder()
	vb, _ := json.Marshal(gin.H{"email": email, "code": otp.Code})
	req2, _ := http.NewRequest("POST", "/auth/otp/verify", bytes.NewBuffer(vb))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)
	Expect(w2.Code).To(Equal(200))
	token := decodeBody(w2.Body)["access_token"].(string)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/users/me", nil)
	req3.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w3, req3)
	Expect(w3.Code).To(Equal(200))
	id := decodeBody(w3.Body)["id"].(string)

	return token, id
}

// connStatusFor returns viewer's connection_status + is_connected for targetID,
// as exposed by the enriched GET /users search endpoint.
func connStatusFor(token, search, targetID string) (string, bool) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users?search="+search, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	body := decodeBody(w.Body)
	items, _ := body["items"].([]interface{})
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok || m["id"] != targetID {
			continue
		}
		status, _ := m["connection_status"].(string)
		connected, _ := m["is_connected"].(bool)
		return status, connected
	}
	return "", false
}

// incomingRequestID returns the id of the PENDING request addressed to the token
// holder whose requester is fromID (empty if none).
func incomingRequestID(token, fromID string) string {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/connections/requests", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	body := decodeBody(w.Body)
	items, _ := body["items"].([]interface{})
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		user, _ := m["user"].(map[string]interface{})
		if user != nil && user["id"] == fromID {
			return m["id"].(string)
		}
	}
	return ""
}

func connectionListIDs(token string) []string {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/connections", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	body := decodeBody(w.Body)
	items, _ := body["items"].([]interface{})
	ids := []string{}
	for _, it := range items {
		if m, ok := it.(map[string]interface{}); ok {
			ids = append(ids, m["id"].(string))
		}
	}
	return ids
}

func connectionsGroup() {
	var (
		tokenA, idA string
		userA       = "conna"
		tokenB, idB string
		userB       = "connb"
		tokenC, idC string
		userC       = "connc"
		tokenD, idD string
		userD       = "connd"
	)

	Describe("Setup", func() {
		It("creates four participants", func() {
			tokenA, idA = registerVerifiedUser("conna@test.com", userA)
			tokenB, idB = registerVerifiedUser("connb@test.com", userB)
			tokenC, idC = registerVerifiedUser("connc@test.com", userC)
			tokenD, idD = registerVerifiedUser("connd@test.com", userD)
			Expect(idA).NotTo(BeEmpty())
			Expect(idB).NotTo(BeEmpty())
		})
	})

	Describe("Sending requests (POST /users/:id/connect)", func() {
		It("A sends a connection request to B", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", idB), nil)
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
		})

		It("rejects connecting to yourself", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", idA), nil)
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("requires authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", idB), nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})

		It("exposes pending_outgoing to the requester", func() {
			status, connected := connStatusFor(tokenA, userB, idB)
			Expect(status).To(Equal("pending_outgoing"))
			Expect(connected).To(BeFalse())
		})

		It("exposes pending_incoming to the addressee", func() {
			status, connected := connStatusFor(tokenB, userA, idA)
			Expect(status).To(Equal("pending_incoming"))
			Expect(connected).To(BeFalse())
		})
	})

	Describe("Incoming requests (GET /connections/requests)", func() {
		It("lists the request addressed to B with the hydrated requester", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/connections/requests", nil)
			req.Header.Set("Authorization", "Bearer "+tokenB)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			body := decodeBody(w.Body)
			Expect(body["total"]).To(BeNumerically(">=", 1))
			Expect(incomingRequestID(tokenB, idA)).NotTo(BeEmpty())
		})

		It("requires authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/connections/requests", nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})

	Describe("Accepting (POST /connections/requests/:id/accept)", func() {
		It("B accepts A's request", func() {
			reqID := incomingRequestID(tokenB, idA)
			Expect(reqID).NotTo(BeEmpty())
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/connections/requests/%s/accept", reqID), nil)
			req.Header.Set("Authorization", "Bearer "+tokenB)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
		})

		It("establishes a connection visible to both sides", func() {
			statusA, connectedA := connStatusFor(tokenA, userB, idB)
			Expect(statusA).To(Equal("connected"))
			Expect(connectedA).To(BeTrue())

			statusB, connectedB := connStatusFor(tokenB, userA, idA)
			Expect(statusB).To(Equal("connected"))
			Expect(connectedB).To(BeTrue())
		})

		It("lists each other under GET /connections", func() {
			Expect(connectionListIDs(tokenA)).To(ContainElement(idB))
			Expect(connectionListIDs(tokenB)).To(ContainElement(idA))
		})

		It("clears the pending request from B's incoming list", func() {
			Expect(incomingRequestID(tokenB, idA)).To(BeEmpty())
		})

		It("rejects an invalid request id", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/connections/requests/not-a-uuid/accept", nil)
			req.Header.Set("Authorization", "Bearer "+tokenB)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Canceling (DELETE /users/:id/connect)", func() {
		It("A sends then cancels a request to C", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", idC), nil)
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))

			status, _ := connStatusFor(tokenA, userC, idC)
			Expect(status).To(Equal("pending_outgoing"))

			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/users/%s/connect", idC), nil)
			req2.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(200))
		})

		It("removes the request from both views", func() {
			statusA, _ := connStatusFor(tokenA, userC, idC)
			Expect(statusA).To(Equal("none"))
			Expect(incomingRequestID(tokenC, idA)).To(BeEmpty())
		})
	})

	Describe("Rejecting (POST /connections/requests/:id/reject)", func() {
		It("A sends a request to D and D rejects it", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", idD), nil)
			req.Header.Set("Authorization", "Bearer "+tokenA)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))

			reqID := incomingRequestID(tokenD, idA)
			Expect(reqID).NotTo(BeEmpty())

			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("POST", fmt.Sprintf("/connections/requests/%s/reject", reqID), nil)
			req2.Header.Set("Authorization", "Bearer "+tokenD)
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(200))
		})

		It("does not establish a connection", func() {
			_, connectedA := connStatusFor(tokenA, userD, idD)
			Expect(connectedA).To(BeFalse())
			Expect(connectionListIDs(tokenA)).NotTo(ContainElement(idD))
		})

		It("clears the request from D's pending incoming list", func() {
			Expect(incomingRequestID(tokenD, idA)).To(BeEmpty())
		})
	})

	Describe("Listing connections (GET /connections)", func() {
		It("requires authentication", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/connections", nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(401))
		})
	})
}
