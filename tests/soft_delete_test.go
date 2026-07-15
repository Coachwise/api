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

// Nothing is really deleted. Deleting sets deleted_at, every read filters it out,
// and the row stays for the refund, the audit and the argument afterwards.
//
// The failure mode this guards against is quiet: miss one `deleted_at IS NULL`
// and deleted rows come back from the dead in that one query. So each case
// checks BOTH halves — gone from the API, still in the table.
// phoneLogin runs the real passwordless flow — request a code, read it out of the
// database, verify it — and returns the access token and user id.
func phoneLogin(phone string) (string, string) {
	// A code was just sent to this number moments ago in test time, and the OTP
	// cooldown would (rightly) refuse another. Age the old ones out instead of
	// waiting.
	db.Exec(`UPDATE otps SET created_at = created_at - interval '1 hour'
	         WHERE user_id IN (SELECT id FROM users WHERE phone = $1)`, phone)

	send := httptest.NewRecorder()
	body, _ := json.Marshal(gin.H{"phone": phone})
	req, _ := http.NewRequest("POST", "/auth/phone/otp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(send, req)
	Expect(send.Code).To(Equal(200), send.Body.String())

	var code string
	Expect(db.Get(&code, `
		SELECT o.code FROM otps o JOIN users u ON u.id = o.user_id
		WHERE u.phone = $1 ORDER BY o.created_at DESC LIMIT 1`, phone)).To(Succeed())

	verify := httptest.NewRecorder()
	vbody, _ := json.Marshal(gin.H{"phone": phone, "code": code})
	vreq, _ := http.NewRequest("POST", "/auth/phone/verify", bytes.NewBuffer(vbody))
	vreq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(verify, vreq)
	Expect(verify.Code).To(Equal(200), verify.Body.String())
	token := decodeBody(verify.Body)["access_token"].(string)

	var id string
	Expect(db.Get(&id, "SELECT id FROM users WHERE phone = $1", phone)).To(Succeed())
	return token, id
}

func createPackage(token, name string) string {
	w := httptest.NewRecorder()
	body, _ := json.Marshal(gin.H{"name": name})
	req, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(201), w.Body.String())
	return decodeBody(w.Body)["id"].(string)
}

func softDeleteGroup() {
	var coachToken string

	BeforeEach(func() {
		if coachToken != "" {
			return
		}
		var coachID string
		coachToken, coachID = registerVerifiedUser("softdel@test.com", "softdeluser")
		makeCoach(coachID)
	})

	It("hides a deleted exercise from every read but keeps the row", func() {
		exID := createExercise(coachToken, "Doomed Exercise")

		del := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/exercises/"+exID, nil)
		req.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(del, req)
		Expect(del.Code).To(Equal(204))

		// Gone from fetch...
		get := httptest.NewRecorder()
		greq, _ := http.NewRequest("GET", "/exercises/"+exID, nil)
		greq.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(get, greq)
		Expect(get.Code).To(Equal(404))

		// ...and gone from the list.
		list := httptest.NewRecorder()
		lreq, _ := http.NewRequest("GET", "/exercises?limit=100", nil)
		lreq.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(list, lreq)
		Expect(list.Code).To(Equal(200))
		for _, item := range decodeBody(list.Body)["items"].([]interface{}) {
			Expect(item.(map[string]interface{})["id"]).NotTo(Equal(exID))
		}

		// But the row is still there, which is the whole point.
		var deletedAt *string
		Expect(db.QueryRow("SELECT deleted_at FROM exercises WHERE id = $1", exID).Scan(&deletedAt)).To(Succeed())
		Expect(deletedAt).NotTo(BeNil(), "the row must survive the delete")
	})

	It("gives a returning phone its own account back, with its history", func() {
		const phone = "+989120000001"

		// Sign up by phone, then delete the account.
		token, userID := phoneLogin(phone)
		del := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(del, req)
		Expect(del.Code).To(Equal(204))

		// The account is gone as far as the app is concerned.
		me := httptest.NewRecorder()
		mreq, _ := http.NewRequest("GET", "/users/me", nil)
		mreq.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(me, mreq)
		Expect(me.Code).To(Equal(401))

		// Come back with the same number: same account, not a new one. This is why
		// the unique constraints on users never had to learn about soft deletes.
		backToken, backID := phoneLogin(phone)
		Expect(backID).To(Equal(userID), "a returning phone must get its own account back")
		Expect(backToken).NotTo(BeEmpty())

		var deletedAt *string
		Expect(db.QueryRow("SELECT deleted_at FROM users WHERE id = $1", userID).Scan(&deletedAt)).To(Succeed())
		Expect(deletedAt).To(BeNil(), "verifying the code revives the account")
	})

	It("keeps a cancelled subscription and lets the client subscribe again", func() {
		// A coach, a package, and a client holding it.
		pkg := createPackage(coachToken, "Cancellable Package")
		clientToken, clientID := registerVerifiedUser("softdelclient@test.com", "softdelclient")
		Expect(clientToken).NotTo(BeEmpty())

		assign := func() int {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"user_id": clientID})
			req, _ := http.NewRequest("POST", "/packages/"+pkg+"/assign", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			return w.Code
		}
		Expect(assign()).To(Equal(200))

		// The coach drops them, with a reason.
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE",
			fmt.Sprintf("/packages/%s/subscribers/%s?reason=not+a+fit", pkg, clientID), nil)
		req.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(200), w.Body.String())

		// The row survives, and says who ended it and why — this is what a refund
		// and a dispute are argued from.
		var status, reason string
		var canceledBy *string
		var deletedAt, endsAt *string
		Expect(db.QueryRow(`
			SELECT status, cancel_reason, canceled_by::text, deleted_at::text, ends_at::text
			FROM package_subscriptions WHERE package_id = $1 AND client_id = $2`,
			pkg, clientID).Scan(&status, &reason, &canceledBy, &deletedAt, &endsAt)).To(Succeed())
		Expect(status).To(Equal("CANCELED"))
		Expect(reason).To(Equal("not a fit"))
		Expect(canceledBy).NotTo(BeNil())
		Expect(deletedAt).NotTo(BeNil())
		Expect(endsAt).NotTo(BeNil(), "the term is what the refund is calculated from")

		// And the cancelled row must not block them from coming back — the unique
		// (package_id, client_id) index would, if enrolling didn't revive the row.
		Expect(assign()).To(Equal(200), "a cancelled client must be able to subscribe again")
	})
}
