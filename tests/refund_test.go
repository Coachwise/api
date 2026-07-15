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

// A coach who drops a paying client pays for it: the client gets back the part of
// the term they won't get. Inside the cooling-off window (escrow_hold_days) the
// money is still held and the whole purchase comes back; after it, only the
// unused days do — taken from the coach's net and our fee in proportion.
//
// These assert against the LEDGER, not just the status code: the point of the
// exercise is that the money ends up in the right wallets.
func refundGroup() {
	const price = int64(1_000_000) // Toman

	// balance is what a wallet is worth, escrow included — the sum of the ledger.
	balance := func(userID string) int64 {
		var v *int64
		Expect(db.QueryRow(`
			SELECT COALESCE(SUM(t.amount), 0) FROM wallet_transactions t
			JOIN wallets w ON w.id = t.wallet_id
			WHERE w.owner_id = $1`, userID).Scan(&v)).To(Succeed())
		if v == nil {
			return 0
		}
		return *v
	}

	// orderTotal is what the client actually paid. Their WALLET nets to zero after
	// a purchase — the buy flow tops it up by exactly the shortfall and spends it —
	// so the wallet is the wrong place to read the price from.
	orderTotal := func(clientID string) int64 {
		var v int64
		Expect(db.QueryRow(`SELECT total FROM orders WHERE buyer_id = $1
			ORDER BY created_at DESC LIMIT 1`, clientID).Scan(&v)).To(Succeed())
		return v
	}

	// buy sets up a coach with a priced package and a client who purchases it.
	buy := func(tag string, months int) (coachToken, coachID, clientID, pkgID string) {
		coachToken, coachID = registerVerifiedUser(tag+"coach@test.com", tag+"coachuser")
		makeCoach(coachID)
		pkgID = createPackage(coachToken, tag+" package")

		// Price it, so the quote has something to work from.
		w := httptest.NewRecorder()
		body, _ := json.Marshal(gin.H{"currency": "IRR", "amount": price})
		req, _ := http.NewRequest("PUT", "/packages/"+pkgID+"/prices", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(BeNumerically("<", 300), w.Body.String())

		clientToken, cid := registerVerifiedUser(tag+"client@test.com", tag+"clientuser")
		clientID = cid

		p := httptest.NewRecorder()
		pbody, _ := json.Marshal(gin.H{"currency": "IRR", "provider": "stub", "months": months})
		preq, _ := http.NewRequest("POST", "/packages/"+pkgID+"/purchase", bytes.NewBuffer(pbody))
		preq.Header.Set("Content-Type", "application/json")
		preq.Header.Set("Authorization", "Bearer "+clientToken)
		router.ServeHTTP(p, preq)
		Expect(p.Code).To(BeNumerically("<", 300), p.Body.String())
		return coachToken, coachID, clientID, pkgID
	}

	cancel := func(coachToken, pkgID, clientID string) gin.H {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE",
			fmt.Sprintf("/packages/%s/subscribers/%s?reason=dropped", pkgID, clientID), nil)
		req.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(200), w.Body.String())
		return decodeBody(w.Body)
	}

	It("refunds the whole purchase when the coach cancels inside the cooling-off window", func() {
		coachToken, coachID, clientID, pkgID := buy("cool", 1)

		// The client paid; the coach is owed their net, still in escrow.
		paid := orderTotal(clientID)
		Expect(paid).To(BeNumerically(">", 0))
		Expect(balance(coachID)).To(BeNumerically(">", 0), "the coach's earnings sit in escrow")

		body := cancel(coachToken, pkgID, clientID)
		refund := body["refund"].(map[string]interface{})
		Expect(refund["full"]).To(BeTrue(), "inside the window the whole purchase comes back")
		Expect(int64(refund["amount"].(float64))).To(Equal(paid))

		// The client is holding every rial back, and the coach kept nothing.
		Expect(balance(clientID)).To(Equal(paid), "the whole purchase is back in the client's wallet")
		Expect(balance(coachID)).To(Equal(int64(0)), "the coach gave back everything they'd earned")

		var status string
		var refunded int64
		Expect(db.QueryRow(`SELECT status, refunded_amount FROM orders
			WHERE buyer_id = $1 ORDER BY created_at DESC LIMIT 1`, clientID).Scan(&status, &refunded)).To(Succeed())
		Expect(status).To(Equal("REFUNDED"))
		Expect(refunded).To(Equal(paid))
	})

	It("refunds only the unused days once the cooling-off window has passed", func() {
		// 3 months: the duration tiers are 1/3/12, so 2 isn't a purchasable term.
		coachToken, coachID, clientID, pkgID := buy("prorata", 3)
		paid := orderTotal(clientID)
		coachEarned := balance(coachID)

		// Age the whole purchase by 45 days: bought then, on a ~91-day term, so about
		// half of it is left and the cooling-off window is long gone. ends_at has to
		// move too — otherwise the term looks 45 days longer than it was sold as.
		db.MustExec(`UPDATE orders SET created_at = now() - interval '45 days' WHERE buyer_id = $1`, clientID)
		db.MustExec(`UPDATE package_subscriptions
		             SET created_at = now() - interval '45 days',
		                 ends_at    = ends_at - interval '45 days'
		             WHERE client_id = $1`, clientID)

		body := cancel(coachToken, pkgID, clientID)
		refund := body["refund"].(map[string]interface{})
		amount := int64(refund["amount"].(float64))

		Expect(refund["full"]).To(BeFalse(), "past the window, the used days are not refunded")
		Expect(amount).To(BeNumerically("<", paid), "they used some of the term")
		Expect(amount).To(BeNumerically(">", 0), "and they didn't use all of it")

		// Roughly half the term was left. Allow slack: months aren't equal lengths.
		Expect(amount).To(BeNumerically("~", paid/2, paid/10))

		// The client is out of pocket only for what they used, and the coach gave
		// back their share of it — never more than they earned.
		Expect(balance(clientID)).To(Equal(amount), "only the unused days come back")
		fromCoach := int64(refund["from_coach"].(float64))
		Expect(fromCoach).To(BeNumerically(">", 0))
		Expect(fromCoach).To(BeNumerically("<=", coachEarned))
		Expect(balance(coachID)).To(Equal(coachEarned - fromCoach))

		var status string
		Expect(db.QueryRow(`SELECT status FROM orders WHERE buyer_id = $1
			ORDER BY created_at DESC LIMIT 1`, clientID).Scan(&status)).To(Succeed())
		Expect(status).To(Equal("PARTIALLY_REFUNDED"))
	})

	It("refuses a payout while a clawback has left the coach owing money", func() {
		coachToken, coachID, clientID, pkgID := buy("negative", 1)

		// Give the coach somewhere to be paid, release the escrow, and let them
		// withdraw the lot — for real, through the API.
		acc := httptest.NewRecorder()
		abody, _ := json.Marshal(gin.H{"card_number": "6037991234567890", "account_holder": "Coach"})
		areq, _ := http.NewRequest("PUT", "/wallet/payout-account", bytes.NewBuffer(abody))
		areq.Header.Set("Content-Type", "application/json")
		areq.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(acc, areq)
		Expect(acc.Code).To(BeNumerically("<", 300), acc.Body.String())

		db.MustExec(`UPDATE wallet_transactions SET available_at = now() - interval '1 day'
		             WHERE wallet_id IN (SELECT id FROM wallets WHERE owner_id = $1)`, coachID)

		earned := balance(coachID)
		Expect(earned).To(BeNumerically(">", 0))

		out := httptest.NewRecorder()
		obody, _ := json.Marshal(gin.H{"currency": "IRR", "amount": earned})
		oreq, _ := http.NewRequest("POST", "/wallet/payout", bytes.NewBuffer(obody))
		oreq.Header.Set("Content-Type", "application/json")
		oreq.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(out, oreq)
		Expect(out.Code).To(Equal(200), out.Body.String())
		Expect(balance(coachID)).To(Equal(int64(0)), "the money has left the wallet")

		// Now they drop the client. The refund claws back money that is already gone.
		cancel(coachToken, pkgID, clientID)

		Expect(balance(coachID)).To(BeNumerically("<", 0), "the coach owes the refund back")

		w := httptest.NewRecorder()
		body, _ := json.Marshal(gin.H{"currency": "IRR", "amount": 1000})
		req, _ := http.NewRequest("POST", "/wallet/payout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(400))
		Expect(decodeBody(w.Body)["code"]).To(BeEquivalentTo(1309), "balance is negative")
	})

	It("has nothing to refund when the coach handed the package over for free", func() {
		coachToken, _, _, pkgID := buy("free", 1)
		clientToken, clientID := registerVerifiedUser("freegift@test.com", "freegiftuser")
		Expect(clientToken).NotTo(BeEmpty())

		w := httptest.NewRecorder()
		body, _ := json.Marshal(gin.H{"user_id": clientID})
		req, _ := http.NewRequest("POST", "/packages/"+pkgID+"/assign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+coachToken)
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(200), w.Body.String())

		// No money changed hands, so cancelling costs nothing — and isn't an error.
		out := cancel(coachToken, pkgID, clientID)
		Expect(out["refund"]).To(BeNil())
		Expect(balance(clientID)).To(Equal(int64(0)))
	})
}
