package llm

// Action catalog: the declarative list of what the agent may propose. This is
// the single source of truth the system prompt is built from and that proposals
// are validated against. The FE holds a matching executor keyed by Name (synced
// like errcode <-> errors.ts). Kind drives policy: read = worker auto-runs it
// (scoped to the user) for grounding; write = client renders an approval card.
// NO wallet and NO delete actions exist here by design — the model can't phrase
// what isn't in the catalog.

type Kind string

const (
	KindRead  Kind = "read"
	KindWrite Kind = "write"
)

type Action struct {
	Name        string
	Domain      string
	Kind        Kind
	Description string
	Params      string // one-line arg hint for the prompt
}

// Action names — the single source of truth referenced by the catalog, the
// server-side read dispatch, and the client executor registry.
const (
	ActionSearchExercises        = "search_exercises"
	ActionGetExercise            = "get_exercise"
	ActionSearchPlans            = "search_plans"
	ActionGetPlan                = "get_plan"
	ActionListAssignedPlans      = "list_assigned_plans"
	ActionListExerciseCategories = "list_exercise_categories"
	ActionListClients            = "list_clients"
	ActionListConnections        = "list_connections"
	ActionSearchUsers            = "search_users"
	ActionSearchTags             = "search_tags"
	ActionGetMe                  = "get_me"

	ActionCreateExercise  = "create_exercise"
	ActionCreatePlan      = "create_plan"
	ActionAddPlanExercise = "add_plan_exercise"
	ActionAssignPlan      = "assign_plan"
)

// catalog grows per phase. Reads run server-side scoped to the user (phase 3);
// writes are executed by the client on approval (phase 4).
var catalog = []Action{
	// Reads — grounding. They run automatically and return results.
	{ActionSearchExercises, "training", KindRead, "Search the exercise library and the user's own exercises.", `{"q": string, "sport"?: "STRENGTH"|"CLIMBING"|"CARDIO"|"MOBILITY"|"GENERAL"}`},
	{ActionGetExercise, "training", KindRead, "Get one exercise by id (name, sport, tracked metrics).", `{"id": uuid}`},
	{ActionSearchPlans, "training", KindRead, "Search the user's training plans by name.", `{"q": string}`},
	{ActionGetPlan, "training", KindRead, "Get a plan with its ordered exercises and set prescriptions.", `{"id": uuid}`},
	{ActionListAssignedPlans, "training", KindRead, "List plans a coach assigned to the user.", `{}`},
	{ActionListExerciseCategories, "training", KindRead, "List exercise categories for a sport (use before create_exercise).", `{"sport"?: string}`},
	{ActionListClients, "coaching", KindRead, "List the coach's clients (empty if the user isn't a coach).", `{}`},
	{ActionListConnections, "social", KindRead, "List the user's accepted connections.", `{}`},
	{ActionSearchUsers, "social", KindRead, "Find users by name or username.", `{"q": string, "coach_only"?: bool}`},
	{ActionSearchTags, "social", KindRead, "Search tags.", `{"q": string, "sport"?: string}`},
	{ActionGetMe, "profile", KindRead, "Get the current user's profile (username, coach status, gender).", `{}`},

	// Writes — proposed only; the user must approve each one.
	{ActionCreateExercise, "training", KindWrite, "Create a personal exercise (always personal). Returns its id.", `{"name": string, "sport": string, "description"?: string}`},
	{ActionCreatePlan, "training", KindWrite, "Create a training plan. Only include exercises whose real id you already have (from search_exercises or a create_exercise result); otherwise create it empty and add exercises after. Returns its id.", `{"name": string, "sport": string, "exercises"?: [{"exercise_id": uuid, "sets"?: [...]}]}`},
	{ActionAddPlanExercise, "training", KindWrite, "Add one existing exercise (by id) to a plan (by id).", `{"plan_id": uuid, "exercise_id": uuid, "sets"?: [{"rep_count"?: int, "duration"?: int, "rest_time": int}]}`},
	{ActionAssignPlan, "training", KindWrite, "Assign a plan to one of the coach's clients.", `{"plan_id": uuid, "client_id": uuid}`},
}

// Catalog returns all registered actions.
func Catalog() []Action { return catalog }

// Lookup finds an action by name.
func Lookup(name string) (Action, bool) {
	for _, a := range catalog {
		if a.Name == name {
			return a, true
		}
	}
	return Action{}, false
}
