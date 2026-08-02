package events

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"coachwise/src/app/models"
	"coachwise/src/database"
	"coachwise/src/llm"
	"coachwise/src/logger"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// SubjectAI carries AIJob messages — one queued assistant turn.
const SubjectAI = "events.ai"

const aiQueueGroup = "ai-workers"

// AIJob asks the worker to produce the assistant reply for MessageID (the
// pending assistant row the view pre-created).
type AIJob struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	MessageID      uuid.UUID `json:"message_id"`
}

// EmitAI queues an assistant turn. Publishes to NATS; if the bus is down it runs
// inline in the background so the reply still lands (dev).
func EmitAI(conversationID, userID, messageID uuid.UUID) {
	job := AIJob{ConversationID: conversationID, UserID: userID, MessageID: messageID}
	if c := getConn(); c != nil && c.IsConnected() {
		if payload, err := json.Marshal(job); err == nil {
			if err := c.Publish(SubjectAI, payload); err == nil {
				return
			}
		}
	}
	go RunAITurn(job)
}

// StartAIConsumer subscribes the worker to assistant-turn jobs.
func StartAIConsumer() {
	c := getConn()
	if c == nil {
		logger.Info("events: AI consumer not started (bus disabled)")
		return
	}
	_, err := c.QueueSubscribe(SubjectAI, aiQueueGroup, func(m *nats.Msg) {
		var job AIJob
		if err := json.Unmarshal(m.Data, &job); err != nil {
			logger.Errorf("events: bad AI job: %v", err)
			return
		}
		RunAITurn(job)
	})
	if err != nil {
		logger.Errorf("events: AI subscribe failed: %v", err)
		return
	}
	logger.Info("events: AI consumer subscribed")
}

// persistedAction is a proposed/executed action stored on the assistant message.
// The client renders write proposals as approval cards and reports the result.
type persistedAction struct {
	Name   string          `json:"name"`
	Args   json.RawMessage `json:"args"`
	Kind   string          `json:"kind"`
	Status string          `json:"status"` // pending | done | failed
	Result json.RawMessage `json:"result,omitempty"`
}

// RunAITurn runs the agent loop for one user turn: prompt the model, auto-run
// any reads (scoped to the user) to ground it, and stop when it proposes writes
// or gives a plain answer. Writes are never executed here — they go to the
// client for approval.
func RunAITurn(job AIJob) {
	if !llm.Enabled() {
		markAIFailed(job)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	conv, err := models.GetConversation(ctx, job.ConversationID, job.UserID)
	if err != nil {
		logger.Errorf("ai: load conversation %s: %v", job.ConversationID, err)
		markAIFailed(job)
		return
	}

	if err := summarizeIfNeeded(ctx, &conv); err != nil {
		logger.Errorf("ai: summarize %s: %v", conv.ID, err) // non-fatal
	}

	base, err := buildContext(ctx, conv, job.MessageID)
	if err != nil {
		logger.Errorf("ai: build context %s: %v", conv.ID, err)
		markAIFailed(job)
		return
	}

	system := llm.BuildSystem()
	if conv.Memory != "" {
		system += "\n\nConversation memory (earlier context):\n" + conv.Memory
	}

	var toolMsgs []llm.Message
	var lastModel string
	var lastUsage llm.Usage

	for i := 0; i < llm.MaxIterations(); i++ {
		resp, err := llm.Complete(ctx, llm.Request{System: system, Messages: append(base, toolMsgs...)})
		if err != nil {
			logger.Errorf("ai: complete %s: %v", conv.ID, err)
			markAIFailed(job)
			return
		}
		lastModel, lastUsage = resp.Model, resp.Usage
		parsed := llm.Parse(resp.Text)
		reads, writes := partitionActions(parsed.Actions)

		// Settle: writes to approve, or a plain answer.
		if len(writes) > 0 || len(reads) == 0 {
			status := models.AIStatusDone
			var actionsJSON []byte
			if len(writes) > 0 {
				status = models.AIStatusAwaiting
				actionsJSON = marshalProposals(writes)
			}
			finalizeTurn(ctx, job, conv.ID, parsed.Text, actionsJSON, status, lastModel, lastUsage)
			return
		}

		// Only reads: run them, feed results back, loop.
		for _, r := range reads {
			toolMsgs = append(toolMsgs, llm.Message{
				Role:    llm.RoleTool,
				Content: r.Name + " => " + executeRead(ctx, job.UserID, r),
			})
		}
	}

	// Never settled within the loop cap → fail; the client localizes.
	markAIFailed(job)
}

func finalizeTurn(ctx context.Context, job AIJob, convID uuid.UUID, text string, actions []byte, status, model string, u llm.Usage) {
	if err := models.UpdateAssistantMessage(ctx, job.MessageID, text, actions, status, model, usageOf(u)); err != nil {
		logger.Errorf("ai: finalize %s: %v", job.MessageID, err)
	}
	_ = models.TouchConversation(ctx, convID)
	EmitSignal(job.UserID, "ai")
}

// markAIFailed flags the pending turn failed (no BE-authored text — the client
// renders a localized message from the failed status).
func markAIFailed(job AIJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = models.FailAIMessage(ctx, job.MessageID)
	EmitSignal(job.UserID, "ai")
}

// buildContext turns the memory-window messages into the model's prompt history,
// skipping the pending assistant row we're about to fill.
func buildContext(ctx context.Context, conv models.AIConversation, pendingID uuid.UUID) ([]llm.Message, error) {
	msgs, err := models.WindowMessages(ctx, conv.ID, conv.SummarizedUntil)
	if err != nil {
		return nil, err
	}
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.ID == pendingID || m.Status == models.AIStatusPending {
			continue
		}
		content := m.Text
		if len(m.Actions) > 2 { // include executed action results so IDs persist
			content += "\n[actions: " + string(m.Actions) + "]"
		}
		role := llm.RoleUser
		if m.Role == models.AIRoleAssistant {
			role = llm.RoleAssistant
		}
		out = append(out, llm.Message{Role: role, Content: content})
	}
	return out, nil
}

// summarizeIfNeeded folds the oldest window turns into the rolling memory once
// the window grows past a threshold, so context stays bounded for a small model.
func summarizeIfNeeded(ctx context.Context, conv *models.AIConversation) error {
	const trigger = 20 // window messages before we compact
	const keepRecent = 8

	msgs, err := models.WindowMessages(ctx, conv.ID, conv.SummarizedUntil)
	if err != nil || len(msgs) <= trigger {
		return err
	}
	fold := msgs[:len(msgs)-keepRecent]

	var b strings.Builder
	if conv.Memory != "" {
		b.WriteString("Previous summary:\n" + conv.Memory + "\n\n")
	}
	b.WriteString("Summarize this coaching conversation. Preserve the user's goals, constraints, decisions, and any IDs created by actions. Be concise.\n\nTranscript:\n")
	for _, m := range fold {
		b.WriteString(m.Role + ": " + m.Text + "\n")
		if len(m.Actions) > 2 {
			b.WriteString("(actions: " + string(m.Actions) + ")\n")
		}
	}

	resp, err := llm.Complete(ctx, llm.Request{Messages: []llm.Message{{Role: llm.RoleUser, Content: b.String()}}})
	if err != nil {
		return err
	}
	cursor := fold[len(fold)-1].CreatedAt
	if err := models.UpdateConversationMemory(ctx, conv.ID, resp.Text, cursor); err != nil {
		return err
	}
	conv.Memory, conv.SummarizedUntil = resp.Text, &cursor
	return nil
}

// partitionActions splits proposed actions by catalog kind, dropping any the
// catalog doesn't know (a hallucinated action never runs).
func partitionActions(actions []llm.ProposedAction) (reads, writes []llm.ProposedAction) {
	for _, a := range actions {
		def, ok := llm.Lookup(a.Name)
		if !ok {
			continue
		}
		switch def.Kind {
		case llm.KindRead:
			reads = append(reads, a)
		case llm.KindWrite:
			writes = append(writes, a)
		}
	}
	return reads, writes
}

func marshalProposals(writes []llm.ProposedAction) []byte {
	out := make([]persistedAction, 0, len(writes))
	for _, w := range writes {
		out = append(out, persistedAction{Name: w.Name, Args: w.Args, Kind: string(llm.KindWrite), Status: "pending"})
	}
	b, _ := json.Marshal(out)
	return b
}

func usageOf(u llm.Usage) models.AIUsage {
	return models.AIUsage{PromptTokens: u.PromptTokens, CompletionTokens: u.CompletionTokens, TotalTokens: u.TotalTokens}
}

// readFunc runs one read action server-side, scoped to the user, returning a
// compact JSON result string for the model. Reads never mutate.
type readFunc func(ctx context.Context, userID uuid.UUID, args json.RawMessage) string

// readHandlers dispatches a read action by name (keyed by the catalog consts, so
// there are no bare string literals and adding a read is one map entry + one fn).
var readHandlers = map[string]readFunc{
	llm.ActionSearchExercises:        readSearchExercises,
	llm.ActionGetExercise:            readGetExercise,
	llm.ActionSearchPlans:            readSearchPlans,
	llm.ActionGetPlan:                readGetPlan,
	llm.ActionListAssignedPlans:      readListAssignedPlans,
	llm.ActionListExerciseCategories: readListExerciseCategories,
	llm.ActionListClients:            readListClients,
	llm.ActionListConnections:        readListConnections,
	llm.ActionSearchUsers:            readSearchUsers,
	llm.ActionSearchTags:             readSearchTags,
	llm.ActionGetMe:                  readGetMe,
}

// executeRead dispatches a read action; visibility is the user's own (the model
// can't see more than the user could).
func executeRead(ctx context.Context, userID uuid.UUID, a llm.ProposedAction) string {
	h, ok := readHandlers[a.Name]
	if !ok {
		return note(noteUnavailable)
	}
	return h(ctx, userID, a.Args)
}

func readSearchExercises(ctx context.Context, userID uuid.UUID, raw json.RawMessage) string {
	var args struct{ Q, Sport string }
	_ = json.Unmarshal(raw, &args)
	items, _, err := models.ListExercisesPaginated(ctx, userID, nil, args.Q, "", args.Sport, database.Paginate{Limit: 8})
	if err != nil {
		return note(noteSearchFail)
	}
	return jm(exRefs(items))
}

func readGetExercise(_ context.Context, userID uuid.UUID, raw json.RawMessage) string {
	id, ok := argID(raw)
	if !ok {
		return note(noteInvalidID)
	}
	e, err := models.GetExrcise(id)
	if err != nil || e == nil || (!e.Public && (e.UserID == nil || *e.UserID != userID)) {
		return note(noteNotFound)
	}
	return jm(exDetailOf(e))
}

func readSearchPlans(ctx context.Context, userID uuid.UUID, raw json.RawMessage) string {
	var args struct{ Q string }
	_ = json.Unmarshal(raw, &args)
	items, _, err := models.ListPlansPaginated(ctx, userID, false, args.Q, database.Paginate{Limit: 8})
	if err != nil {
		return note(noteSearchFail)
	}
	return jm(planRefs(items))
}

func readGetPlan(ctx context.Context, userID uuid.UUID, raw json.RawMessage) string {
	id, ok := argID(raw)
	if !ok {
		return note(noteInvalidID)
	}
	p, err := models.GetPlanForUser(ctx, id, userID)
	if err != nil || p == nil {
		return note(noteNotFound)
	}
	exs, _ := models.ListPlanExercises(ctx, id)
	type pex struct {
		Order    int             `json:"order"`
		Exercise json.RawMessage `json:"exercise"`
		Sets     json.RawMessage `json:"sets"`
	}
	out := make([]pex, 0, len(exs))
	for _, pe := range exs {
		out = append(out, pex{pe.ExerciseOrder, json.RawMessage(pe.Exercise), json.RawMessage(pe.Sets)})
	}
	return jm(map[string]any{"plan": planRefOf(*p), "exercises": out})
}

func readListAssignedPlans(ctx context.Context, userID uuid.UUID, _ json.RawMessage) string {
	items, err := models.ListUserAssignedPlans(ctx, userID)
	if err != nil {
		return note(noteFailed)
	}
	return jm(planRefs(items))
}

func readListExerciseCategories(_ context.Context, _ uuid.UUID, raw json.RawMessage) string {
	var args struct{ Sport string }
	_ = json.Unmarshal(raw, &args)
	cats, err := models.ListExerciseCategories(args.Sport)
	if err != nil {
		return note(noteFailed)
	}
	type catRef struct {
		ID   uuid.UUID `json:"id"`
		Slug string    `json:"slug"`
		Name any       `json:"name"`
	}
	refs := make([]catRef, 0, len(cats))
	for _, c := range cats {
		refs = append(refs, catRef{c.ID, c.Slug, c.NameI18n})
	}
	return jm(refs)
}

func readListClients(ctx context.Context, userID uuid.UUID, _ json.RawMessage) string {
	clients, _, err := models.ListCoachClients(ctx, userID, database.Paginate{Limit: 25})
	if err != nil {
		return note(noteFailed)
	}
	refs := make([]userRef, 0, len(clients))
	for _, c := range clients {
		refs = append(refs, userRefOf(c.User))
	}
	return jm(refs)
}

func readListConnections(ctx context.Context, userID uuid.UUID, _ json.RawMessage) string {
	users, _, err := models.ListConnectionsPaginated(ctx, userID, database.Paginate{Limit: 25})
	if err != nil {
		return note(noteFailed)
	}
	return jm(userRefs(users))
}

func readSearchUsers(ctx context.Context, userID uuid.UUID, raw json.RawMessage) string {
	var args struct {
		Q         string `json:"q"`
		CoachOnly bool   `json:"coach_only"`
	}
	_ = json.Unmarshal(raw, &args)
	users, _, err := models.ListUsersPaginated(ctx, args.Q, args.CoachOnly, nil, userID, database.Paginate{Limit: 8})
	if err != nil {
		return note(noteSearchFail)
	}
	return jm(userRefs(users))
}

func readSearchTags(ctx context.Context, _ uuid.UUID, raw json.RawMessage) string {
	var args struct{ Q, Sport string }
	_ = json.Unmarshal(raw, &args)
	tags, _, err := models.SearchTags(ctx, args.Q, args.Sport, 8, 0)
	if err != nil {
		return note(noteSearchFail)
	}
	type tagRef struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	refs := make([]tagRef, 0, len(tags))
	for _, t := range tags {
		refs = append(refs, tagRef{t.ID, t.Name})
	}
	return jm(refs)
}

func readGetMe(_ context.Context, userID uuid.UUID, _ json.RawMessage) string {
	u, err := models.GetUser(userID)
	if err != nil || u == nil {
		return note(noteFailed)
	}
	r := userRefOf(*u)
	return jm(map[string]any{"id": r.ID, "username": r.Username, "name": r.Name, "is_coach": r.IsCoach, "gender": u.Gender})
}

// --- read-result serializers (compact: id + label, small for the model) ------

// note strings are model-facing (English, like the rest of the BE), returned as
// a read result the model reads — not user-facing errors.
const (
	noteUnavailable = "not available"
	noteFailed      = "failed"
	noteSearchFail  = "search failed"
	noteNotFound    = "not found"
	noteInvalidID   = "invalid id"
)

func jm(v any) string { b, _ := json.Marshal(v); return string(b) }

func note(s string) string { return jm(map[string]string{"note": s}) }

func argID(raw json.RawMessage) (uuid.UUID, bool) {
	var a struct{ ID string }
	if json.Unmarshal(raw, &a) != nil {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(a.ID)
	return id, err == nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type exRef struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Sport  string    `json:"sport"`
	Public bool      `json:"public"`
}

func exRefOf(e models.Exercise) exRef {
	return exRef{e.ID, e.Name, string(e.SportType), e.Public}
}

func exRefs(items []models.Exercise) []exRef {
	out := make([]exRef, 0, len(items))
	for _, e := range items {
		out = append(out, exRefOf(e))
	}
	return out
}

func exDetailOf(e *models.Exercise) map[string]any {
	return map[string]any{
		"id": e.ID, "name": e.Name, "description": e.Description, "sport": string(e.SportType), "public": e.Public,
		"tracks": map[string]bool{"weight": e.TrackWeight, "distance": e.TrackDistance, "grade": e.TrackGrade, "height": e.TrackHeight},
	}
}

type planRef struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Exercises        int       `json:"exercises"`
	EstimatedSeconds int       `json:"estimated_seconds"`
}

func planRefOf(p models.Plan) planRef {
	return planRef{p.ID, p.Name, p.ExerciseCount, p.EstimatedSeconds}
}

func planRefs(items []models.Plan) []planRef {
	out := make([]planRef, 0, len(items))
	for _, p := range items {
		out = append(out, planRefOf(p))
	}
	return out
}

type userRef struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Name     string    `json:"name"`
	IsCoach  bool      `json:"is_coach"`
}

func userRefOf(u models.User) userRef {
	name := strings.TrimSpace(deref(u.FirstName) + " " + deref(u.LastName))
	return userRef{u.ID, u.Username, name, u.IsCoach}
}

func userRefs(items []models.User) []userRef {
	out := make([]userRef, 0, len(items))
	for _, u := range items {
		out = append(out, userRefOf(u))
	}
	return out
}
