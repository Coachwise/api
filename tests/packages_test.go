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

// makeCoach flips a user into a coach by inserting into the coaches table, which
// fires the trigger that sets users.is_coach.
func makeCoach(userID string) {
	_, err := db.Exec(
		"INSERT INTO coaches (user_id, specialties) VALUES ($1, ARRAY['FITNESS']::sports[]) ON CONFLICT (user_id) DO NOTHING",
		userID,
	)
	Expect(err).NotTo(HaveOccurred())
}

// createOwnedPlan creates a plan owned by the token holder and returns its id.
func createOwnedPlan(token, name string) string {
	w := httptest.NewRecorder()
	body, _ := json.Marshal(gin.H{"name": name, "public": false})
	req, _ := http.NewRequest("POST", "/plans", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(201))
	return decodeBody(w.Body)["id"].(string)
}

func packagesGroup() {
	var (
		coachToken, coachID   string
		clientToken, clientID string
		otherToken, otherID   string
		planA, planB          string
		packageID             string
	)

	Describe("Setup", func() {
		It("creates a coach, a client, and a non-coach", func() {
			coachToken, coachID = registerVerifiedUser("pkgcoach@test.com", "pkgcoach")
			clientToken, clientID = registerVerifiedUser("pkgclient@test.com", "pkgclient")
			otherToken, otherID = registerVerifiedUser("pkgother@test.com", "pkgother")
			makeCoach(coachID)
			planA = createOwnedPlan(coachToken, "Push Plan")
			planB = createOwnedPlan(coachToken, "Pull Plan")
			Expect(coachID).NotTo(BeEmpty())
			Expect(clientID).NotTo(BeEmpty())
		})
	})

	Describe("Creating packages (POST /packages)", func() {
		It("rejects non-coaches with 403", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"name": "Nope"})
			req, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(403))
		})

		It("requires a name", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"price_monthly": 100})
			req, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("rejects bundling a plan the coach does not own", func() {
			// otherPlan is owned by the non-coach user.
			otherPlan := createOwnedPlan(otherToken, "Foreign Plan")
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"name": "Bad Bundle", "plan_ids": []string{otherPlan}})
			req, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})

		It("creates a package bundling owned plans", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name":            "Premium Monthly",
				"description":     "Full coaching",
				"price_monthly":   500000,
				"trial_days":      7,
				"video_access":    true,
				"custom_features": []string{"weekly check-in", "nutrition"},
				"plan_ids":        []string{planA, planB},
			})
			req, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
			b := decodeBody(w.Body)
			packageID = b["id"].(string)
			Expect(b["plan_count"]).To(BeEquivalentTo(2))
			Expect(b["is_active"]).To(Equal(true))
			plans, _ := b["plans"].([]interface{})
			Expect(plans).To(HaveLen(2))
		})
	})

	Describe("Listing & fetching", func() {
		It("lists the coach's packages", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/packages", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			b := decodeBody(w.Body)
			Expect(b["total"]).To(BeEquivalentTo(1))
		})

		It("fetches a single package", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/packages/"+packageID, nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["name"]).To(Equal("Premium Monthly"))
		})

		It("hides another coach's package fetch (404)", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/packages/"+packageID, nil)
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})
	})

	Describe("Updating (PUT /packages/:id)", func() {
		It("updates fields and bundled plans", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{
				"name":      "Premium Renamed",
				"is_active": false,
				"plan_ids":  []string{planA},
			})
			req, _ := http.NewRequest("PUT", "/packages/"+packageID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			b := decodeBody(w.Body)
			Expect(b["name"]).To(Equal("Premium Renamed"))
			Expect(b["is_active"]).To(Equal(false))
			Expect(b["plan_count"]).To(BeEquivalentTo(1))
		})
	})

	Describe("Setting plans (PUT /packages/:id/plans)", func() {
		It("replaces the bundled plan set", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"plan_ids": []string{planA, planB}})
			req, _ := http.NewRequest("PUT", "/packages/"+packageID+"/plans", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["plan_count"]).To(BeEquivalentTo(2))
		})
	})

	Describe("Assigning (POST /packages/:id/assign)", func() {
		It("assigns every bundled plan to the client", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"user_id": clientID})
			req, _ := http.NewRequest("POST", "/packages/"+packageID+"/assign", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))

			// The client should now see both plans in their assigned plans list.
			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("GET", "/plans", nil)
			req2.Header.Set("Authorization", "Bearer "+clientToken)
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(200))
			items, _ := decodeBody(w2.Body)["items"].([]interface{})
			Expect(len(items)).To(BeNumerically(">=", 2))
		})
	})

	Describe("Coach clients (GET /coaches/clients)", func() {
		It("lists connected clients with the plans the coach assigned", func() {
			// Connect coach <-> client so the client is in the coach's network.
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", clientID), nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))

			reqID := incomingRequestID(clientToken, coachID)
			Expect(reqID).NotTo(BeEmpty())
			wa := httptest.NewRecorder()
			reqa, _ := http.NewRequest("POST", "/connections/requests/"+reqID+"/accept", nil)
			reqa.Header.Set("Authorization", "Bearer "+clientToken)
			router.ServeHTTP(wa, reqa)
			Expect(wa.Code).To(Equal(200))

			wc := httptest.NewRecorder()
			reqc, _ := http.NewRequest("GET", "/coaches/clients", nil)
			reqc.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(wc, reqc)
			Expect(wc.Code).To(Equal(200))
			items, _ := decodeBody(wc.Body)["items"].([]interface{})
			Expect(items).NotTo(BeEmpty())
			found := false
			for _, it := range items {
				m := it.(map[string]interface{})
				if m["id"] == clientID {
					found = true
					assigned, _ := m["assigned_plans"].([]interface{})
					Expect(len(assigned)).To(BeNumerically(">=", 2))
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Describe("Public coach packages (GET /coaches/:id/packages)", func() {
		It("returns only active packages", func() {
			// Reactivate first so it surfaces.
			wp := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"name": "Premium Renamed", "is_active": true})
			reqp, _ := http.NewRequest("PUT", "/packages/"+packageID, bytes.NewBuffer(body))
			reqp.Header.Set("Content-Type", "application/json")
			reqp.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(wp, reqp)
			Expect(wp.Code).To(Equal(200))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/coaches/"+coachID+"/packages", nil)
			req.Header.Set("Authorization", "Bearer "+clientToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(decodeBody(w.Body)["total"]).To(BeEquivalentTo(1))
		})
	})

	Describe("Client model (subscription-based, not connection-based)", func() {
		clientIDs := func(token string) []string {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/coaches/clients", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			ids := []string{}
			for _, it := range items {
				ids = append(ids, it.(map[string]interface{})["id"].(string))
			}
			return ids
		}

		It("does not list a mere connection as a client", func() {
			// otherUser only connects with the coach — never enrolls in a package.
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", fmt.Sprintf("/users/%s/connect", otherID), nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			reqID := incomingRequestID(otherToken, coachID)
			wa := httptest.NewRecorder()
			reqa, _ := http.NewRequest("POST", "/connections/requests/"+reqID+"/accept", nil)
			reqa.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(wa, reqa)
			Expect(wa.Code).To(Equal(200))

			Expect(clientIDs(coachToken)).NotTo(ContainElement(otherID))
		})

		It("lists a user as a client after they subscribe to a package", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/packages/"+packageID+"/subscribe", nil)
			req.Header.Set("Authorization", "Bearer "+otherToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))

			Expect(clientIDs(coachToken)).To(ContainElement(otherID))
		})

		It("rejects subscribing to your own package", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/packages/"+packageID+"/subscribe", nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(400))
		})
	})

	Describe("Constraints (no dup plans, one package per client)", func() {
		clientPlanCount := func(token string) int {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/plans", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			return len(items)
		}

		It("does not duplicate a plan when assigned again", func() {
			before := clientPlanCount(clientToken) // client already has planA+planB from the package
			// Assign planA directly again.
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"user_id": clientID})
			req, _ := http.NewRequest("POST", "/plans/"+planA+"/assign", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(200))
			Expect(clientPlanCount(clientToken)).To(Equal(before)) // unchanged — no duplicate
		})

		It("rejects assigning a second (different) package to the same client", func() {
			// Create a second package.
			w := httptest.NewRecorder()
			body, _ := json.Marshal(gin.H{"name": "Second Package", "plan_ids": []string{planA}})
			req, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(201))
			pkg2 := decodeBody(w.Body)["id"].(string)

			// clientID already holds packageID — assigning pkg2 must fail.
			w2 := httptest.NewRecorder()
			body2, _ := json.Marshal(gin.H{"user_id": clientID})
			req2, _ := http.NewRequest("POST", "/packages/"+pkg2+"/assign", bytes.NewBuffer(body2))
			req2.Header.Set("Content-Type", "application/json")
			req2.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(400))
		})
	})

	Describe("Unsubscribe (DELETE /packages/:id/subscribers/:user_id)", func() {
		// Count only non-public plans — for a fresh client that owns none, those are
		// exactly the plans assigned to them (GET /plans also returns public plans).
		clientPlanCount := func(token string) int {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/plans", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)
			items, _ := decodeBody(w.Body)["items"].([]interface{})
			n := 0
			for _, it := range items {
				if m, ok := it.(map[string]interface{}); ok {
					if pub, _ := m["public"].(bool); !pub {
						n++
					}
				}
			}
			return n
		}

		It("removes package-assigned plans but keeps manually-assigned ones", func() {
			unsubToken, unsubID := registerVerifiedUser("pkgunsub@test.com", "pkgunsub")

			// Make the package bundle exactly planA + planB.
			wp := httptest.NewRecorder()
			pb, _ := json.Marshal(gin.H{"plan_ids": []string{planA, planB}})
			rp, _ := http.NewRequest("PUT", "/packages/"+packageID+"/plans", bytes.NewBuffer(pb))
			rp.Header.Set("Content-Type", "application/json")
			rp.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(wp, rp)
			Expect(wp.Code).To(Equal(200))

			// Assign the package (stamps planA, planB with the package id).
			wa := httptest.NewRecorder()
			ab, _ := json.Marshal(gin.H{"user_id": unsubID})
			ra, _ := http.NewRequest("POST", "/packages/"+packageID+"/assign", bytes.NewBuffer(ab))
			ra.Header.Set("Content-Type", "application/json")
			ra.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(wa, ra)
			Expect(wa.Code).To(Equal(200))

			// Manually assign a separate plan (package_id stays NULL).
			manualPlan := createOwnedPlan(coachToken, "Manual Extra")
			wm := httptest.NewRecorder()
			mb, _ := json.Marshal(gin.H{"user_id": unsubID})
			rm, _ := http.NewRequest("POST", "/plans/"+manualPlan+"/assign", bytes.NewBuffer(mb))
			rm.Header.Set("Content-Type", "application/json")
			rm.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(wm, rm)
			Expect(wm.Code).To(Equal(200))

			Expect(clientPlanCount(unsubToken)).To(Equal(3)) // planA, planB (package) + manual

			// Unsubscribe: the two package plans go, the manual one stays.
			wd := httptest.NewRecorder()
			rd, _ := http.NewRequest("DELETE", "/packages/"+packageID+"/subscribers/"+unsubID, nil)
			rd.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(wd, rd)
			Expect(wd.Code).To(Equal(200))

			Expect(clientPlanCount(unsubToken)).To(Equal(1)) // only the manual plan remains
		})
	})

	Describe("Deleting (DELETE /packages/:id)", func() {
		It("deletes the package", func() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/packages/"+packageID, nil)
			req.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(204))

			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest("GET", "/packages/"+packageID, nil)
			req2.Header.Set("Authorization", "Bearer "+coachToken)
			router.ServeHTTP(w2, req2)
			Expect(w2.Code).To(Equal(404))
		})
	})
}
