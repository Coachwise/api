package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"coachwise/src/app/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Support tickets are a turn-based conversation: each side must wait for the
// other to answer. The admin answers from the admin panel (a direct DB write,
// simulated here), and a worker loop turns that reply into a user notification.
// These tests assert the turn lock, the worker's atomic hand-off, and ownership
// isolation.
func supportGroup() {
	var (
		token   string
		userID  string
		token2  string
		userID2 string
	)

	// do issues an authenticated JSON request and returns the recorder.
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

	// adminReplies does what the admin panel's Reply action does: writes an ADMIN
	// message straight to the DB (delivered_at NULL) and hands the turn back.
	adminReplies := func(ticketID, body string) {
		_, err := db.Exec(`INSERT INTO support_messages (ticket_id, sender, body) VALUES ($1, 'ADMIN', $2)`, ticketID, body)
		Expect(err).To(BeNil())
		_, err = db.Exec(`UPDATE support_tickets SET turn = 'USER', last_message_at = now() WHERE id = $1`, ticketID)
		Expect(err).To(BeNil())
	}

	BeforeEach(func() {
		if token != "" {
			return
		}
		token, userID = registerVerifiedUser("support1@test.com", "supuser1")
		token2, userID2 = registerVerifiedUser("support2@test.com", "supuser2")
	})

	It("opens a ticket awaiting the admin, with the first message stored", func() {
		w := do(http.MethodPost, "/support/tickets", token, gin.H{
			"subject": "Charged twice",
			"body":    "I was billed two times for one package.",
		})
		Expect(w.Code).To(Equal(http.StatusCreated), w.Body.String())
		body := decodeBody(w.Body)
		ticket := body["ticket"].(map[string]any)
		Expect(ticket["turn"]).To(Equal("ADMIN"))
		Expect(ticket["status"]).To(Equal("OPEN"))
		msg := body["message"].(map[string]any)
		Expect(msg["sender"]).To(Equal("USER"))
	})

	It("blocks the user from sending twice before the admin answers", func() {
		w := do(http.MethodPost, "/support/tickets", token, gin.H{"subject": "Turn test", "body": "first"})
		ticketID := decodeBody(w.Body)["ticket"].(map[string]any)["id"].(string)

		// It is the admin's turn now — a second user message must be refused.
		w2 := do(http.MethodPost, "/support/tickets/"+ticketID+"/messages", token, gin.H{"body": "any update?"})
		Expect(w2.Code).To(Equal(http.StatusConflict), w2.Body.String())
		Expect(decodeBody(w2.Body)["code"]).To(BeEquivalentTo(1206))
	})

	It("lets the user reply once the admin has answered, then blocks again", func() {
		w := do(http.MethodPost, "/support/tickets", token, gin.H{"subject": "Ping pong", "body": "hello"})
		ticketID := decodeBody(w.Body)["ticket"].(map[string]any)["id"].(string)

		adminReplies(ticketID, "Hi, how can I help?")

		// User's turn now → reply accepted.
		w2 := do(http.MethodPost, "/support/tickets/"+ticketID+"/messages", token, gin.H{"body": "my card was double charged"})
		Expect(w2.Code).To(Equal(http.StatusCreated), w2.Body.String())

		// Back to the admin's turn → blocked again.
		w3 := do(http.MethodPost, "/support/tickets/"+ticketID+"/messages", token, gin.H{"body": "still there?"})
		Expect(w3.Code).To(Equal(http.StatusConflict), w3.Body.String())

		// The thread is in order: user, admin, user.
		w4 := do(http.MethodGet, "/support/tickets/"+ticketID, token, nil)
		msgs := decodeBody(w4.Body)["messages"].([]any)
		Expect(msgs).To(HaveLen(3))
		Expect(msgs[0].(map[string]any)["sender"]).To(Equal("USER"))
		Expect(msgs[1].(map[string]any)["sender"]).To(Equal("ADMIN"))
		Expect(msgs[2].(map[string]any)["sender"]).To(Equal("USER"))
	})

	It("delivers an admin reply exactly once via the worker's atomic claim", func() {
		w := do(http.MethodPost, "/support/tickets", token, gin.H{"subject": "Delivery", "body": "help"})
		ticketID := decodeBody(w.Body)["ticket"].(map[string]any)["id"].(string)
		adminReplies(ticketID, "Sorted, refunded.")

		ctx := context.Background()
		first, err := models.ClaimUndeliveredReplies(ctx)
		Expect(err).To(BeNil())

		// Exactly this ticket's reply is claimed, addressed to the opener.
		var mine *models.SupportDelivery
		for i := range first {
			if first[i].TicketID.String() == ticketID {
				mine = &first[i]
			}
		}
		Expect(mine).NotTo(BeNil(), "the admin reply should be claimed once")
		Expect(mine.UserID.String()).To(Equal(userID))

		// A second claim must not re-deliver it (delivered_at is now set).
		second, err := models.ClaimUndeliveredReplies(ctx)
		Expect(err).To(BeNil())
		for i := range second {
			Expect(second[i].TicketID.String()).NotTo(Equal(ticketID), "already delivered, must not repeat")
		}
	})

	It("never leaks a ticket to another user", func() {
		w := do(http.MethodPost, "/support/tickets", token, gin.H{"subject": "Private", "body": "secret"})
		ticketID := decodeBody(w.Body)["ticket"].(map[string]any)["id"].(string)

		// A different user cannot read it...
		w2 := do(http.MethodGet, "/support/tickets/"+ticketID, token2, nil)
		Expect(w2.Code).To(Equal(http.StatusNotFound))

		// ...nor post to it.
		w3 := do(http.MethodPost, "/support/tickets/"+ticketID+"/messages", token2, gin.H{"body": "intruding"})
		Expect(w3.Code).To(Equal(http.StatusNotFound))

		// ...and it is absent from their ticket list.
		w4 := do(http.MethodGet, "/support/tickets", token2, nil)
		items := decodeBody(w4.Body)["items"].([]any)
		for _, it := range items {
			Expect(it.(map[string]any)["id"]).NotTo(Equal(ticketID))
		}
		_ = userID2
	})

	It("rejects a reply to a closed ticket", func() {
		w := do(http.MethodPost, "/support/tickets", token, gin.H{"subject": "Closing", "body": "hi"})
		ticketID := decodeBody(w.Body)["ticket"].(map[string]any)["id"].(string)
		adminReplies(ticketID, "answer") // hands the turn back to the user
		_, err := db.Exec(`UPDATE support_tickets SET status = 'CLOSED' WHERE id = $1`, ticketID)
		Expect(err).To(BeNil())

		w2 := do(http.MethodPost, "/support/tickets/"+ticketID+"/messages", token, gin.H{"body": "wait"})
		Expect(w2.Code).To(Equal(http.StatusConflict))
		Expect(decodeBody(w2.Body)["code"]).To(BeEquivalentTo(1207))
	})

	It("404s an unknown ticket id", func() {
		w := do(http.MethodGet, "/support/tickets/"+uuid.NewString(), token, nil)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
}
