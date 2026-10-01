package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PithomLabs/ebp/internal/domain"
)

// DebtRepository handles debt item persistence.
type DebtRepository struct{}

// NewDebtRepository creates a new DebtRepository.
func NewDebtRepository() *DebtRepository {
	return &DebtRepository{}
}

// AddFullDebt inserts the six canonical debt items for a new idea.
// Idempotent: skips items already present.
func (r *DebtRepository) AddFullDebt(ctx context.Context, q Querier, ideaID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range domain.AllDebtItems {
		_, err := q.ExecContext(ctx,
			`INSERT OR IGNORE INTO ebp_debt_item (id, idea_id, item, added_at, retired)
			 VALUES (?, ?, ?, ?, 0)`,
			generateUUID(), ideaID, item, now)
		if err != nil {
			return fmt.Errorf("insert debt item %s: %w", item, err)
		}
	}
	return nil
}

// AddOne adds a single debt item. If the item exists and is retired,
// it is un-retired (new evidence creates new debt).
func (r *DebtRepository) AddOne(ctx context.Context, q Querier, ideaID, item string) error {
	if !domain.IsValidDebtItem(item) {
		return ErrInvalidDebtItem
	}

	// Check if item already exists
	var exists int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ebp_debt_item WHERE idea_id = ? AND item = ?`,
		ideaID, item).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check debt existence: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if exists == 0 {
		// New item — insert
		_, err := q.ExecContext(ctx,
			`INSERT INTO ebp_debt_item (id, idea_id, item, added_at, retired)
			 VALUES (?, ?, ?, ?, 0)`,
			generateUUID(), ideaID, item, now)
		if err != nil {
			return fmt.Errorf("add debt item: %w", err)
		}
	} else {
		// Item exists — if retired, un-retire it (new evidence creates new debt)
		_, err := q.ExecContext(ctx,
			`UPDATE ebp_debt_item SET retired = 0, retired_at = NULL, evidence = NULL
			 WHERE idea_id = ? AND item = ? AND retired = 1`,
			ideaID, item)
		if err != nil {
			return fmt.Errorf("un-retire debt item: %w", err)
		}
	}
	return nil
}

// RetireOne retires exactly one debt item with evidence.
func (r *DebtRepository) RetireOne(ctx context.Context, q Querier, ideaID, item, evidence string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := q.ExecContext(ctx,
		`UPDATE ebp_debt_item SET retired = 1, retired_at = ?, evidence = ?
		 WHERE idea_id = ? AND item = ? AND retired = 0`,
		now, evidence, ideaID, item)
	if err != nil {
		return fmt.Errorf("retire debt item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		var exists int
		err := q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM ebp_debt_item WHERE idea_id = ? AND item = ?`,
			ideaID, item).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check debt existence: %w", err)
		}
		if exists == 0 {
			return ErrDebtNotPresent
		}
		return ErrDebtAlreadyRetired
	}
	return nil
}

// ListByIdea returns all debt items for an idea.
func (r *DebtRepository) ListByIdea(ctx context.Context, q Querier, ideaID string) ([]domain.DebtItem, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, idea_id, item, added_at, retired, retired_at, evidence
		 FROM ebp_debt_item WHERE idea_id = ? ORDER BY added_at`, ideaID)
	if err != nil {
		return nil, fmt.Errorf("list debt items: %w", err)
	}
	defer rows.Close()

	var items []domain.DebtItem
	for rows.Next() {
		var d domain.DebtItem
		var retired int
		var retiredAt, evidence sql.NullString
		if err := rows.Scan(&d.ID, &d.IdeaID, &d.Item, &d.AddedAt, &retired, &retiredAt, &evidence); err != nil {
			return nil, fmt.Errorf("scan debt item: %w", err)
		}
		d.Retired = retired != 0
		d.RetiredAt = retiredAt.String
		d.Evidence = evidence.String
		items = append(items, d)
	}
	return items, rows.Err()
}

// CountOpen returns the number of unretired debt items for an idea.
func (r *DebtRepository) CountOpen(ctx context.Context, q Querier, ideaID string) (int, error) {
	var count int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ebp_debt_item WHERE idea_id = ? AND retired = 0`, ideaID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count open debt: %w", err)
	}
	return count, nil
}
