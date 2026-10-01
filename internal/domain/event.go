package domain

// Event action constants — append-only audit log.
const (
	EventCaptured           = "captured"
	EventDebtAdded          = "debt.added"
	EventDebtRetired        = "debt.retired"
	EventPromoted           = "promoted"
	EventFinalTruthFlagged  = "final_truth.flagged"
	EventFinalTruthCleared  = "final_truth.cleared"
)

// Event is an append-only audit log entry.
type Event struct {
	ID        string `json:"id"`
	IdeaID    string `json:"idea_id"`
	Action    string `json:"action"`
	Detail    string `json:"detail,omitempty"`
	Actor     string `json:"actor,omitempty"`
	CreatedAt string `json:"created_at"`
}
