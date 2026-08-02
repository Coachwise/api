package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"coachwise/src/llm"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeLLM is a deterministic in-test provider (the sanctioned no-mock path): it
// proposes a read first, then a write once results come back — exercising the
// whole agent loop without a live model.
type fakeLLM struct{}

func (fakeLLM) Name() string { return "fake" }

func (fakeLLM) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	hasTool, lastUser := false, ""
	for _, m := range req.Messages {
		switch m.Role {
		case llm.RoleTool:
			hasTool = true
		case llm.RoleUser:
			lastUser = m.Content
		}
	}

	// "readtest" branch: fire several read actions at once, then settle plainly.
	if strings.Contains(lastUser, "readtest") {
		if hasTool {
			return aiResp("SUMMARY_DONE"), nil
		}
		return aiResp("checking a few things:\n" +
			"```action\n{\"name\":\"get_me\"}\n```\n" +
			"```action\n{\"name\":\"search_plans\",\"args\":{\"q\":\"\"}}\n```\n" +
			"```action\n{\"name\":\"list_connections\"}\n```"), nil
	}

	if hasTool {
		return aiResp("proposing an exercise:\n```action\n" +
			`{"name":"create_exercise","args":{"name":"Dead Hang","sport":"CLIMBING"}}` + "\n```"), nil
	}
	return aiResp("let me look:\n```action\n" +
		`{"name":"search_exercises","args":{"q":"hang"}}` + "\n```"), nil
}

func aiResp(text string) llm.Response {
	return llm.Response{Text: text, Model: "fake", Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}
}

func aiPost(token, path string, body gin.H) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	return w
}

func aiConversation(token, id string) gin.H {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ai/conversations/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	Expect(w.Code).To(Equal(200))
	return decodeBody(w.Body)
}

// lastAssistant returns the newest assistant message in a conversation, or nil.
func lastAssistant(token, id string) gin.H {
	msgs, _ := aiConversation(token, id)["messages"].([]interface{})
	var last gin.H
	for _, m := range msgs {
		mm := gin.H(m.(map[string]interface{}))
		if mm["role"] == "assistant" {
			last = mm
		}
	}
	return last
}

func aiFlowGroup() {
	var token, convID string

	Describe("AI assistant", func() {
		It("injects a deterministic provider", func() {
			llm.SetProvider(fakeLLM{})
			token, _ = registerVerifiedUser("aiuser@test.com", "aiuser")
		})

		It("starts a conversation and queues an assistant turn", func() {
			w := aiPost(token, "/ai/conversations", gin.H{"text": "make a hang exercise"})
			Expect(w.Code).To(Equal(201))
			body := decodeBody(w.Body)
			conv := body["conversation"].(map[string]interface{})
			convID = conv["id"].(string)
			Expect(convID).NotTo(BeEmpty())
			// The assistant reply starts pending (the worker fills it async).
			msg := body["message"].(map[string]interface{})
			Expect(msg["status"]).To(Equal("pending"))
		})

		It("runs the loop: read then a write proposal awaiting approval", func() {
			Eventually(func() string {
				m := lastAssistant(token, convID)
				if m == nil {
					return ""
				}
				return m["status"].(string)
			}, 5*time.Second, 100*time.Millisecond).Should(Equal("awaiting_approval"))

			m := lastAssistant(token, convID)
			actions := m["actions"].([]interface{})
			Expect(actions).To(HaveLen(1))
			Expect(actions[0].(map[string]interface{})["name"]).To(Equal("create_exercise"))
			Expect(actions[0].(map[string]interface{})["kind"]).To(Equal("write"))
			// Token usage is recorded.
			Expect(m["total_tokens"]).To(BeNumerically("==", 15))
		})

		It("continues after a client-reported write result", func() {
			m := lastAssistant(token, convID)
			mid := m["id"].(string)
			w := aiPost(token, "/ai/conversations/"+convID+"/messages/"+mid+"/result",
				gin.H{"results": []gin.H{{"ok": true, "result": gin.H{"id": "created-123"}}}})
			Expect(w.Code).To(Equal(202))

			// A fresh assistant turn is queued and settles again.
			Eventually(func() bool {
				msgs, _ := aiConversation(token, convID)["messages"].([]interface{})
				assistants := 0
				for _, mm := range msgs {
					if gin.H(mm.(map[string]interface{}))["role"] == "assistant" {
						assistants++
					}
				}
				return assistants >= 2
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
		})

		It("runs several read actions server-side and settles to a plain answer", func() {
			w := aiPost(token, "/ai/conversations", gin.H{"text": "readtest my plans"})
			Expect(w.Code).To(Equal(201))
			cid := decodeBody(w.Body)["conversation"].(map[string]interface{})["id"].(string)

			Eventually(func() string {
				m := lastAssistant(token, cid)
				if m == nil {
					return ""
				}
				return m["status"].(string)
			}, 5*time.Second, 100*time.Millisecond).Should(Equal("done"))

			m := lastAssistant(token, cid)
			Expect(m["text"]).To(Equal("SUMMARY_DONE"))
			Expect(m["actions"]).To(HaveLen(0)) // reads leave no proposals to approve
		})

		It("rejects access to another user's conversation", func() {
			other, _ := registerVerifiedUser("aiother@test.com", "aiother")
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/ai/conversations/"+convID, nil)
			req.Header.Set("Authorization", "Bearer "+other)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(404))
		})
	})
}
