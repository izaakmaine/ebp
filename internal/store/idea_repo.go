package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PithomLabs/ebp/internal/domain"
)

// IdeaRepository handles idea persistence.
type IdeaRepository struct{}

// NewIdeaRepository creates a new IdeaRepository.
func NewIdeaRepository() *IdeaRepository {
	return &IdeaRepository{}
}

// Create inserts a new idea. The idea must start in proposed/unpromoted state.
func (r *IdeaRepository) Create(ctx context.Context, q Querier, idea *domain.Idea) error {
	if idea.ID == "" {
		return fmt.Errorf("idea ID is required")
	}
	if idea.Owner == "" {
		return fmt.Errorf("idea owner is required")
	}
	if idea.Claim == "" {
		return fmt.Errorf("idea claim is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	idea.Born = now
	idea.CreatedAt = now
	idea.UpdatedAt = now
	idea.Promoted = false
	idea.ContainsFinalTruthClaim = false

	_, err := q.ExecContext(ctx,
		`INSERT INTO ebp_idea (id, owner, claim, source, born, contains_final_truth_claim, promoted, promoted_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		idea.ID, idea.Owner, idea.Claim, nullString(idea.Source), idea.Born,
		boolToInt(idea.ContainsFinalTruthClaim), boolToInt(idea.Promoted),
		nullString(idea.PromotedAt), idea.CreatedAt, idea.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert idea: %w", err)
	}
	return nil
}

// GetByID retrieves an idea by ID.
func (r *IdeaRepository) GetByID(ctx context.Context, q Querier, id string) (*domain.Idea, error) {
	idea := &domain.Idea{}
	var source, promotedAt sql.NullString
	var promoted, ftClaim int
	err := q.QueryRowContext(ctx,
		`SELECT id, owner, claim, source, born, contains_final_truth_claim, promoted, promoted_at, created_at, updated_at
		 FROM ebp_idea WHERE id = ?`, id).Scan(
		&idea.ID, &idea.Owner, &idea.Claim, &source, &idea.Born,
		&ftClaim, &promoted, &promotedAt, &idea.CreatedAt, &idea.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get idea: %w", err)
	}
	idea.Source = source.String
	idea.ContainsFinalTruthClaim = ftClaim != 0
	idea.Promoted = promoted != 0
	idea.PromotedAt = promotedAt.String
	return idea, nil
}

// Update modifies all mutable fields of an idea.
func (r *IdeaRepository) Update(ctx context.Context, q Querier, idea *domain.Idea) error {
	idea.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	result, err := q.ExecContext(ctx,
		`UPDATE ebp_idea SET contains_final_truth_claim = ?, promoted = ?, promoted_at = ?, updated_at = ? WHERE id = ?`,
		boolToInt(idea.ContainsFinalTruthClaim), boolToInt(idea.Promoted),
		nullString(idea.PromotedAt), idea.UpdatedAt, idea.ID)
	if err != nil {
		return fmt.Errorf("update idea: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePromotion modifies only promotion fields (promoted, promoted_at).
// Does not touch contains_final_truth_claim, preventing concurrent overwrites.
func (r *IdeaRepository) UpdatePromotion(ctx context.Context, q Querier, idea *domain.Idea) error {
	idea.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	result, err := q.ExecContext(ctx,
		`UPDATE ebp_idea SET promoted = ?, promoted_at = ?, updated_at = ? WHERE id = ?`,
		boolToInt(idea.Promoted), nullString(idea.PromotedAt), idea.UpdatedAt, idea.ID)
	if err != nil {
		return fmt.Errorf("update promotion: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateFinalTruthClaim modifies only the contains_final_truth_claim field.
// Does not touch promoted/promoted_at, preventing concurrent overwrites.
func (r *IdeaRepository) UpdateFinalTruthClaim(ctx context.Context, q Querier, idea *domain.Idea) error {
	idea.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	result, err := q.ExecContext(ctx,
		`UPDATE ebp_idea SET contains_final_truth_claim = ?, updated_at = ? WHERE id = ?`,
		boolToInt(idea.ContainsFinalTruthClaim), idea.UpdatedAt, idea.ID)
	if err != nil {
		return fmt.Errorf("update final truth claim: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
