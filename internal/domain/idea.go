package domain

// Debt item constants — the six canonical EBP 2.1 debt categories.
const (
	DebtNeedMap              = "needMap"
	DebtNeedInvariant        = "needInvariant"
	DebtNeedToyCheck         = "needToyCheck"
	DebtNeedNullModel        = "needNullModel"
	DebtNeedObstruction      = "needObstruction"
	DebtNeedFaithfulnessReview = "needFaithfulnessReview"
)

// AllDebtItems is the full starting debt for a fresh idea.
var AllDebtItems = []string{
	DebtNeedMap,
	DebtNeedInvariant,
	DebtNeedToyCheck,
	DebtNeedNullModel,
	DebtNeedObstruction,
	DebtNeedFaithfulnessReview,
}

// IsValidDebtItem reports whether item is one of the six canonical debt categories.
func IsValidDebtItem(item string) bool {
	for _, d := range AllDebtItems {
		if d == item {
			return true
		}
	}
	return false
}

// Idea represents an EBP 2.1 epistemic idea.
type Idea struct {
	ID                     string `json:"id"`
	Owner                  string `json:"owner"`
	Claim                  string `json:"claim"`
	Source                 string `json:"source,omitempty"`
	Born                   string `json:"born"`
	ContainsFinalTruthClaim bool   `json:"contains_final_truth_claim"`
	Promoted               bool   `json:"promoted"`
	PromotedAt             string `json:"promoted_at,omitempty"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

// DebtItem represents a single debt obligation on an idea.
type DebtItem struct {
	ID        string `json:"id"`
	IdeaID    string `json:"idea_id"`
	Item      string `json:"item"`
	AddedAt   string `json:"added_at"`
	Retired   bool   `json:"retired"`
	RetiredAt string `json:"retired_at,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
}

// IsPromoted returns whether the idea meets the EBP 2.1 promotion criteria:
// debt is empty AND containsFinalTruthClaim is false.
// This is a derived check; the durable promoted state is stored on the Idea.
func IsPromoted(idea *Idea, openDebtCount int) bool {
	return openDebtCount == 0 && !idea.ContainsFinalTruthClaim
}
