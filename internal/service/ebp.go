package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PithomLabs/ebp/internal/domain"
	"github.com/PithomLabs/ebp/internal/store"
)

// EBP implements the seven canonical EBP 2.1 operations.
type EBP struct {
	ideas  *store.IdeaRepository
	debts  *store.DebtRepository
	events *store.EventRepository
	db     *store.DB
}

// New creates a new EBP service.
func New(db *store.DB) *EBP {
	return &EBP{
		ideas:  store.NewIdeaRepository(),
		debts:  store.NewDebtRepository(),
		events: store.NewEventRepository(),
		db:     db,
	}
}

// Capture creates a new idea with full starting debt.
// Minimum required: owner + claim.
func (e *EBP) Capture(ctx context.Context, owner, claim, source, actor string) (*domain.Idea, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	if claim == "" {
		return nil, fmt.Errorf("claim is required")
	}

	idea := &domain.Idea{
		ID:     generateUUID(),
		Owner:  owner,
		Claim:  claim,
		Source: source,
	}

	if err := e.db.Transaction(func(tx *store.DB) error {
		if err := e.ideas.Create(ctx, tx, idea); err != nil {
			return err
		}
		if err := e.debts.AddFullDebt(ctx, tx, idea.ID); err != nil {
			return err
		}
		detail, _ := json.Marshal(map[string]string{
			"owner":  owner,
			"claim":  claim,
			"source": source,
		})
		return e.events.Append(ctx, tx, &domain.Event{
			IdeaID: idea.ID,
			Action: domain.EventCaptured,
			Detail: string(detail),
			Actor:  actor,
		})
	}); err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}

	return idea, nil
}

// Status returns the current idea state and its debt items from a consistent snapshot.
func (e *EBP) Status(ctx context.Context, ideaID string) (*domain.Idea, []domain.DebtItem, error) {
	var idea *domain.Idea
	var debt []domain.DebtItem

	err := e.db.Transaction(func(tx *store.DB) error {
		var err error
		idea, err = e.ideas.GetByID(ctx, tx, ideaID)
		if err != nil {
			return err
		}
		debt, err = e.debts.ListByIdea(ctx, tx, ideaID)
		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("status: %w", err)
	}
	return idea, debt, nil
}

// Retire retires exactly one debt item with evidence.
func (e *EBP) Retire(ctx context.Context, ideaID, item, evidence, actor string) error {
	if !domain.IsValidDebtItem(item) {
		return store.ErrInvalidDebtItem
	}

	if err := e.db.Transaction(func(tx *store.DB) error {
		if err := e.debts.RetireOne(ctx, tx, ideaID, item, evidence); err != nil {
			return err
		}
		detail, _ := json.Marshal(map[string]string{
			"item":     item,
			"evidence": evidence,
		})
		return e.events.Append(ctx, tx, &domain.Event{
			IdeaID: ideaID,
			Action: domain.EventDebtRetired,
			Detail: string(detail),
			Actor:  actor,
		})
	}); err != nil {
		return fmt.Errorf("retire: %w", err)
	}
	return nil
}

// AddDebt adds one canonical debt item to an idea.
// If the idea was promoted, new debt un-promotes it.
// Reads current state inside the transaction to prevent concurrent promotion races.
func (e *EBP) AddDebt(ctx context.Context, ideaID, item, actor string) error {
	if !domain.IsValidDebtItem(item) {
		return store.ErrInvalidDebtItem
	}

	if err := e.db.Transaction(func(tx *store.DB) error {
		// Read current state inside transaction to get fresh wasPromoted
		idea, err := e.ideas.GetByID(ctx, tx, ideaID)
		if err != nil {
			return err
		}

		if err := e.debts.AddOne(ctx, tx, ideaID, item); err != nil {
			return err
		}

		// New evidence creates new debt — un-promote if currently promoted
		if idea.Promoted {
			idea.Promoted = false
			idea.PromotedAt = ""
			if err := e.ideas.UpdatePromotion(ctx, tx, idea); err != nil {
				return err
			}
		}

		detail, _ := json.Marshal(map[string]string{"item": item})
		return e.events.Append(ctx, tx, &domain.Event{
			IdeaID: ideaID,
			Action: domain.EventDebtAdded,
			Detail: string(detail),
			Actor:  actor,
		})
	}); err != nil {
		return fmt.Errorf("add debt: %w", err)
	}
	return nil
}

// Promote records a promotion event. Succeeds only when debt is empty,
// containsFinalTruthClaim is false, and idea is not already promoted.
// All reads happen inside the transaction to prevent concurrent races.
func (e *EBP) Promote(ctx context.Context, ideaID, actor string) error {
	if err := e.db.Transaction(func(tx *store.DB) error {
		// Read current state inside transaction
		idea, err := e.ideas.GetByID(ctx, tx, ideaID)
		if err != nil {
			return err
		}

		if idea.Promoted {
			return store.ErrPromotionBlocked
		}

		openDebt, err := e.debts.CountOpen(ctx, tx, ideaID)
		if err != nil {
			return err
		}

		if openDebt > 0 || idea.ContainsFinalTruthClaim {
			return store.ErrPromotionBlocked
		}

		now := time.Now().UTC().Format(time.RFC3339)
		idea.Promoted = true
		idea.PromotedAt = now

		// Use targeted update — only touches promoted fields, not contains_final_truth_claim
		if err := e.ideas.UpdatePromotion(ctx, tx, idea); err != nil {
			return err
		}

		return e.events.Append(ctx, tx, &domain.Event{
			IdeaID: ideaID,
			Action: domain.EventPromoted,
			Actor:  actor,
		})
	}); err != nil {
		return fmt.Errorf("promote: %w", err)
	}
	return nil
}

// MarkFinalTruthClaim sets containsFinalTruthClaim = true on an idea.
// Reads current state inside the transaction and uses targeted update.
// Fails if the idea is already promoted (mutually exclusive with promotion).
func (e *EBP) MarkFinalTruthClaim(ctx context.Context, ideaID, actor string) error {
	if err := e.db.Transaction(func(tx *store.DB) error {
		idea, err := e.ideas.GetByID(ctx, tx, ideaID)
		if err != nil {
			return err
		}

		if idea.Promoted {
			return store.ErrPromotionBlocked
		}

		idea.ContainsFinalTruthClaim = true

		// Use targeted update — only touches contains_final_truth_claim, not promoted
		if err := e.ideas.UpdateFinalTruthClaim(ctx, tx, idea); err != nil {
			return err
		}

		return e.events.Append(ctx, tx, &domain.Event{
			IdeaID: ideaID,
			Action: domain.EventFinalTruthFlagged,
			Actor:  actor,
		})
	}); err != nil {
		return fmt.Errorf("mark final truth: %w", err)
	}
	return nil
}

// ClearFinalTruthClaim sets containsFinalTruthClaim = false on an idea.
// Reads current state inside the transaction and uses targeted update.
func (e *EBP) ClearFinalTruthClaim(ctx context.Context, ideaID, actor string) error {
	if err := e.db.Transaction(func(tx *store.DB) error {
		idea, err := e.ideas.GetByID(ctx, tx, ideaID)
		if err != nil {
			return err
		}

		idea.ContainsFinalTruthClaim = false

		// Use targeted update — only touches contains_final_truth_claim, not promoted
		if err := e.ideas.UpdateFinalTruthClaim(ctx, tx, idea); err != nil {
			return err
		}

		return e.events.Append(ctx, tx, &domain.Event{
			IdeaID: ideaID,
			Action: domain.EventFinalTruthCleared,
			Actor:  actor,
		})
	}); err != nil {
		return fmt.Errorf("clear final truth: %w", err)
	}
	return nil
}

func generateUUID() string {
	return domain.GenerateUUID()
}
