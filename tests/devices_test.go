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

// Push tokens identify a device, not a session: the same phone can outlive many
// logins and change hands. These tests cover the upsert, the hand-over to a new
// owner, and unregistering on logout.
func devicesGroup() {
	var (
		token   string
		userID  string
		token2  string
		userID2 string
	)

	do := func(method, path, tok string, payload gin.H) *httptest.ResponseRecorder {
		var buf *bytes.Buffer
		if payload != nil {
			b, _ := json.Marshal(payload)
			buf = bytes.NewBuffer(b)
		} else {
			buf = bytes.NewBuffer(nil)
		}
		req, _ := http.NewRequest(method, path, buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	countFor := func(deviceToken string) int {
		var n int
		Expect(db.Get(&n, "SELECT count(*) FROM device_tokens WHERE token = $1", deviceToken)).To(Succeed())
		return n
	}

	ownerOf := func(deviceToken string) string {
		var owner string
		Expect(db.Get(&owner, "SELECT user_id::text FROM device_tokens WHERE token = $1", deviceToken)).To(Succeed())
		return owner
	}

	BeforeEach(func() {
		if token != "" {
			return
		}
		token, userID = registerVerifiedUser("device1@test.com", "devuser1")
		token2, userID2 = registerVerifiedUser("device2@test.com", "devuser2")
	})

	It("registers a token and stays at one row when the app re-registers", func() {
		body := gin.H{"token": "fcm-token-aaa", "platform": "android", "locale": "fa"}
		Expect(do(http.MethodPost, "/devices", token, body).Code).To(Equal(http.StatusOK))
		Expect(do(http.MethodPost, "/devices", token, body).Code).To(Equal(http.StatusOK))

		Expect(countFor("fcm-token-aaa")).To(Equal(1))
		Expect(ownerOf("fcm-token-aaa")).To(Equal(userID))
	})

	It("rejects an unknown platform", func() {
		w := do(http.MethodPost, "/devices", token, gin.H{"token": "fcm-token-bad", "platform": "windows"})
		Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
		Expect(countFor("fcm-token-bad")).To(Equal(0))
	})

	It("hands a token to the second account that registers it", func() {
		shared := gin.H{"token": "fcm-token-shared", "platform": "ios"}
		Expect(do(http.MethodPost, "/devices", token, shared).Code).To(Equal(http.StatusOK))
		Expect(do(http.MethodPost, "/devices", token2, shared).Code).To(Equal(http.StatusOK))

		// One device, one row — now owned by whoever registered last.
		Expect(countFor("fcm-token-shared")).To(Equal(1))
		Expect(ownerOf("fcm-token-shared")).To(Equal(userID2))
	})

	It("unregisters only the caller's own token", func() {
		mine := gin.H{"token": "fcm-token-mine", "platform": "android"}
		Expect(do(http.MethodPost, "/devices", token, mine).Code).To(Equal(http.StatusOK))

		// Another user naming that token must not be able to silence the device.
		Expect(do(http.MethodDelete, "/devices", token2, gin.H{"token": "fcm-token-mine"}).Code).To(Equal(http.StatusOK))
		Expect(countFor("fcm-token-mine")).To(Equal(1))

		Expect(do(http.MethodDelete, "/devices", token, gin.H{"token": "fcm-token-mine"}).Code).To(Equal(http.StatusOK))
		Expect(countFor("fcm-token-mine")).To(Equal(0))
	})
}
