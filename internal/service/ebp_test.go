package service

import (
	"context"
	"errors"
	"testing"

	"github.com/PithomLabs/ebp/internal/domain"
	"github.com/PithomLabs/ebp/internal/store"
)

func setupTest(t *testing.T) *EBP {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestCaptureCreatesValidIdea(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test claim", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if idea.Owner != "Alice" {
		t.Errorf("owner = %q, want %q", idea.Owner, "Alice")
	}
	if idea.Claim != "test claim" {
		t.Errorf("claim = %q, want %q", idea.Claim, "test claim")
	}
	if idea.Promoted {
		t.Error("new idea should not be promoted")
	}
	if idea.ContainsFinalTruthClaim {
		t.Error("new idea should not contain final truth claim")
	}

	// Verify full debt was attached
	_, debt, err := ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(debt) != 6 {
		t.Fatalf("debt count = %d, want 6", len(debt))
	}
	for _, d := range debt {
		if d.Retired {
			t.Errorf("debt %s should not be retired", d.Item)
		}
	}
}

func TestCaptureRequiresOwnerAndClaim(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	_, err := ebp.Capture(ctx, "", "claim", "", "actor")
	if err == nil {
		t.Error("expected error for empty owner")
	}

	_, err = ebp.Capture(ctx, "Alice", "", "", "actor")
	if err == nil {
		t.Error("expected error for empty claim")
	}
}

func TestAddDebtAddsExactlyOneItem(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	if err := ebp.AddDebt(ctx, idea.ID, domain.DebtNeedObstruction, "alice"); err != nil {
		t.Fatalf("add debt: %v", err)
	}

	// Idempotent: adding same debt again should not fail
	if err := ebp.AddDebt(ctx, idea.ID, domain.DebtNeedObstruction, "alice"); err != nil {
		t.Fatalf("add debt (idempotent): %v", err)
	}

	_, debt, err := ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	// Idempotent: still 6 open items (needObstruction was already present)
	openCount := 0
	for _, d := range debt {
		if !d.Retired {
			openCount++
		}
	}
	if openCount != 6 {
		t.Errorf("open debt count = %d, want 6", openCount)
	}
}

func TestAddDebtInvalidItem(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	err = ebp.AddDebt(ctx, idea.ID, "invalidDebt", "alice")
	if !errors.Is(err, store.ErrInvalidDebtItem) {
		t.Errorf("expected ErrInvalidDebtItem, got %v", err)
	}
}

func TestRetireRemovesExactlyOneItem(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	if err := ebp.Retire(ctx, idea.ID, domain.DebtNeedInvariant, "Invariant: I(B)", "alice"); err != nil {
		t.Fatalf("retire: %v", err)
	}

	_, debt, err := ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	// Exactly 5 should remain open
	openCount := 0
	for _, d := range debt {
		if !d.Retired {
			openCount++
		}
		if d.Item == domain.DebtNeedInvariant && !d.Retired {
			t.Error("needInvariant should be retired")
		}
		if d.Item == domain.DebtNeedInvariant && d.Evidence != "Invariant: I(B)" {
			t.Errorf("evidence = %q, want %q", d.Evidence, "Invariant: I(B)")
		}
	}
	if openCount != 5 {
		t.Errorf("open debt count = %d, want 5", openCount)
	}
}

func TestRetireNonexistentDebtFails(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	err = ebp.Retire(ctx, idea.ID, domain.DebtNeedMap, "evidence", "alice")
	if err != nil {
		t.Fatalf("retire existing debt should succeed: %v", err)
	}

	// Retire again — should fail
	err = ebp.Retire(ctx, idea.ID, domain.DebtNeedMap, "evidence", "alice")
	if !errors.Is(err, store.ErrDebtAlreadyRetired) {
		t.Errorf("expected ErrDebtAlreadyRetired, got %v", err)
	}

	// Retire something that was never added
	err = ebp.Retire(ctx, idea.ID, "needNonexistent", "evidence", "alice")
	if !errors.Is(err, store.ErrInvalidDebtItem) && !errors.Is(err, store.ErrDebtNotPresent) {
		t.Errorf("expected ErrInvalidDebtItem or ErrDebtNotPresent, got %v", err)
	}
}

func TestPromoteFailsWithOutstandingDebt(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	err = ebp.Promote(ctx, idea.ID, "alice")
	if !errors.Is(err, store.ErrPromotionBlocked) {
		t.Errorf("expected ErrPromotionBlocked, got %v", err)
	}
}

func TestPromoteFailsForFinalTruthClaim(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "This proves the final theory", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	// Retire all debt
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, idea.ID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}

	// Flag final truth
	if err := ebp.MarkFinalTruthClaim(ctx, idea.ID, "alice"); err != nil {
		t.Fatalf("mark final truth: %v", err)
	}

	// Should fail
	err = ebp.Promote(ctx, idea.ID, "alice")
	if !errors.Is(err, store.ErrPromotionBlocked) {
		t.Errorf("expected ErrPromotionBlocked, got %v", err)
	}
}

func TestPromoteSucceedsWhenReady(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "candidate bridge", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	// Retire all debt
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, idea.ID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}

	// Promote should succeed
	if err := ebp.Promote(ctx, idea.ID, "alice"); err != nil {
		t.Fatalf("promote: %v", err)
	}

	idea, _, err = ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !idea.Promoted {
		t.Error("idea should be promoted")
	}
	if idea.PromotedAt == "" {
		t.Error("promoted_at should be set")
	}
}

func TestNewDebtAfterPromotion(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	// Retire all debt and promote
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, idea.ID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}
	if err := ebp.Promote(ctx, idea.ID, "alice"); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Add new debt — idea should be un-promoted
	if err := ebp.AddDebt(ctx, idea.ID, domain.DebtNeedObstruction, "bob"); err != nil {
		t.Fatalf("add debt: %v", err)
	}

	idea, debt, err := ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if idea.Promoted {
		t.Error("idea should not be promoted after new debt")
	}
	openCount := 0
	for _, d := range debt {
		if !d.Retired {
			openCount++
		}
	}
	if openCount != 1 {
		t.Errorf("open debt count = %d, want 1", openCount)
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	ebp := New(db)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "persistent claim", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// Retire one debt
	if err := ebp.Retire(ctx, ideaID, domain.DebtNeedMap, "map defined", "alice"); err != nil {
		t.Fatalf("retire: %v", err)
	}

	// Simulate restart: close and reopen
	db.Close()

	db2, err := store.OpenTest()
	if err != nil {
		t.Fatalf("reopen test db: %v", err)
	}
	defer db2.Close()

	// NOTE: OpenTest creates a fresh in-memory DB. In a real file-based DB,
	// the data would persist. This test validates the schema and write path.
	// We test actual persistence with a temp file below.
}

func TestStateSurvivesRestartWithFile(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"

	db, err := store.Open(tmpFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	ebp := New(db)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "persistent claim", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	if err := ebp.Retire(ctx, ideaID, domain.DebtNeedMap, "map defined", "alice"); err != nil {
		t.Fatalf("retire: %v", err)
	}

	db.Close()

	// Reopen
	db2, err := store.Open(tmpFile)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db2.Close()

	ebp2 := New(db2)
	idea2, debt, err := ebp2.Status(ctx, ideaID)
	if err != nil {
		t.Fatalf("status after restart: %v", err)
	}
	if idea2.Claim != "persistent claim" {
		t.Errorf("claim = %q, want %q", idea2.Claim, "persistent claim")
	}

	retiredCount := 0
	for _, d := range debt {
		if d.Retired {
			retiredCount++
		}
	}
	if retiredCount != 1 {
		t.Errorf("retired count = %d, want 1", retiredCount)
	}
}

func TestRepeatedOperationsDeterministic(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	// Double promote should fail
	for _, item := range domain.AllDebtItems {
		ebp.Retire(ctx, idea.ID, item, "e", "alice")
	}
	if err := ebp.Promote(ctx, idea.ID, "alice"); err != nil {
		t.Fatalf("first promote: %v", err)
	}
	err = ebp.Promote(ctx, idea.ID, "alice")
	if !errors.Is(err, store.ErrPromotionBlocked) {
		t.Errorf("second promote should fail, got %v", err)
	}
}

func TestConcurrentMutationSafety(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	// Run concurrent retire attempts on the same debt item
	type result struct {
		err error
	}
	done := make(chan result, 2)

	go func() {
		done <- result{ebp.Retire(ctx, idea.ID, domain.DebtNeedMap, "e1", "agent-1")}
	}()
	go func() {
		done <- result{ebp.Retire(ctx, idea.ID, domain.DebtNeedMap, "e2", "agent-2")}
	}()

	r1 := <-done
	r2 := <-done

	// Exactly one should succeed
	succeeded := 0
	if r1.err == nil {
		succeeded++
	}
	if r2.err == nil {
		succeeded++
	}
	if succeeded != 1 {
		t.Errorf("expected exactly 1 success, got %d (errors: %v, %v)", succeeded, r1.err, r2.err)
	}
}

func TestMarkAndClearFinalTruthClaim(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	if err := ebp.MarkFinalTruthClaim(ctx, idea.ID, "alice"); err != nil {
		t.Fatalf("mark: %v", err)
	}

	idea, _, err = ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !idea.ContainsFinalTruthClaim {
		t.Error("should contain final truth claim")
	}

	if err := ebp.ClearFinalTruthClaim(ctx, idea.ID, "alice"); err != nil {
		t.Fatalf("clear: %v", err)
	}

	idea, _, err = ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if idea.ContainsFinalTruthClaim {
		t.Error("should not contain final truth claim after clear")
	}
}

func TestStatusReturnsCurrentState(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "test claim", "source info", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	got, debt, err := ebp.Status(ctx, idea.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if got.Owner != "Alice" {
		t.Errorf("owner = %q, want %q", got.Owner, "Alice")
	}
	if got.Claim != "test claim" {
		t.Errorf("claim = %q, want %q", got.Claim, "test claim")
	}
	if got.Source != "source info" {
		t.Errorf("source = %q, want %q", got.Source, "source info")
	}
	if got.Promoted {
		t.Error("should not be promoted")
	}
	if len(debt) != 6 {
		t.Errorf("debt count = %d, want 6", len(debt))
	}
}

func TestEventLogRecordsActions(t *testing.T) {
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	ebp := New(db)
	ctx := context.Background()
	events := store.NewEventRepository()

	idea, err := ebp.Capture(ctx, "Alice", "test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	eventList, err := events.ListByIdea(ctx, db, idea.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(eventList) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventList))
	}
	if eventList[0].Action != domain.EventCaptured {
		t.Errorf("event action = %q, want %q", eventList[0].Action, domain.EventCaptured)
	}
}

func TestConcurrentPromoteAndAddDebt(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "concurrent test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// Retire all debt so promote is possible
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, ideaID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}

	const iterations = 50
	for i := 0; i < iterations; i++ {
		type result struct{ err error }
		done := make(chan result, 2)

		go func() { done <- result{ebp.Promote(ctx, ideaID, "promoter")} }()
		go func() { done <- result{ebp.AddDebt(ctx, ideaID, domain.DebtNeedObstruction, "debter")} }()

		r1, r2 := <-done, <-done
		// At least one must not race-crash
		if r1.err != nil && r2.err != nil {
			// Both failed is fine (e.g. blocked), as long as not corrupted
		}

		// Core invariant: never promoted with open debt
		got, debt, err := ebp.Status(ctx, ideaID)
		if err != nil {
			t.Fatalf("iteration %d status: %v", i, err)
		}
		openDebt := 0
		for _, d := range debt {
			if !d.Retired {
				openDebt++
			}
		}
		if got.Promoted && openDebt > 0 {
			t.Fatalf("iteration %d: promoted with %d open debt items", i, openDebt)
		}
	}
}

func TestConcurrentPromoteAndMarkFinalTruth(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "concurrent ft test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// Retire all debt so promote is possible
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, ideaID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}

	const iterations = 50
	for i := 0; i < iterations; i++ {
		// Reset: clear final truth and un-promote between iterations
		ebp.ClearFinalTruthClaim(ctx, ideaID, "reset")

		type result struct{ err error }
		done := make(chan result, 2)

		go func() { done <- result{ebp.Promote(ctx, ideaID, "promoter")} }()
		go func() { done <- result{ebp.MarkFinalTruthClaim(ctx, ideaID, "marker")} }()

		<-done
		<-done

		// Core invariant: never promoted with containsFinalTruthClaim
		got, _, err := ebp.Status(ctx, ideaID)
		if err != nil {
			t.Fatalf("iteration %d status: %v", i, err)
		}
		if got.Promoted && got.ContainsFinalTruthClaim {
			t.Fatalf("iteration %d: promoted AND final-truth-claim both true", i)
		}
	}
}

func TestConcurrentAddDebtAfterPromotion(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "concurrent promote", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// Retire all debt and promote
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, ideaID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}
	if err := ebp.Promote(ctx, ideaID, "alice"); err != nil {
		t.Fatalf("initial promote: %v", err)
	}

	const iterations = 50
	for i := 0; i < iterations; i++ {
		type result struct{ err error }
		done := make(chan result, 2)

		go func() { done <- result{ebp.AddDebt(ctx, ideaID, domain.DebtNeedObstruction, "debter")} }()
		go func() { done <- result{ebp.Promote(ctx, ideaID, "promoter")} }()

		<-done
		<-done

		// Core invariant: adding new debt must leave idea unpromoted
		got, debt, err := ebp.Status(ctx, ideaID)
		if err != nil {
			t.Fatalf("iteration %d status: %v", i, err)
		}
		openDebt := 0
		for _, d := range debt {
			if !d.Retired {
				openDebt++
			}
		}
		if got.Promoted && openDebt > 0 {
			t.Fatalf("iteration %d: promoted with %d open debt items", i, openDebt)
		}
	}
}

func TestStatusConsistencyUnderMutation(t *testing.T) {
	ebp := setupTest(t)
	ctx := context.Background()

	idea, err := ebp.Capture(ctx, "Alice", "status test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// First retire all debt, promote
	for _, item := range domain.AllDebtItems {
		if err := ebp.Retire(ctx, ideaID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}
	if err := ebp.Promote(ctx, ideaID, "alice"); err != nil {
		t.Fatalf("promote: %v", err)
	}

	const iterations = 100
	for i := 0; i < iterations; i++ {
		// Un-promote for next iteration
		if err := ebp.AddDebt(ctx, ideaID, domain.DebtNeedObstruction, "reset"); err != nil {
			t.Fatalf("iteration %d reset: %v", i, err)
		}

		type statusResult struct {
			idea *domain.Idea
			debt []domain.DebtItem
			err  error
		}
		done := make(chan statusResult, 1)

		go func() {
			got, debt, err := ebp.Status(ctx, ideaID)
			done <- statusResult{got, debt, err}
		}()

		// Mutate concurrently
		ebp.Retire(ctx, ideaID, domain.DebtNeedObstruction, "e", "agent")

		s := <-done
		if s.err != nil {
			t.Fatalf("iteration %d status: %v", i, s.err)
		}

		// If promoted, debt must be empty (consistency check)
		if s.idea.Promoted {
			for _, d := range s.debt {
				if !d.Retired {
					t.Fatalf("iteration %d: promoted but has open debt %s", i, d.Item)
				}
			}
		}
	}
}

func TestTransactionPanicRollback(t *testing.T) {
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create a valid idea first
	ebp := New(db)
	idea, err := ebp.Capture(ctx, "Alice", "panic test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// Transaction that panics should roll back without corrupting DB
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to be caught")
			}
		}()
		_ = db.Transaction(func(tx *store.DB) error {
			panic("boom")
		})
	}()

	// DB must still be usable and the idea must still exist
	got, _, err := ebp.Status(ctx, ideaID)
	if err != nil {
		t.Fatalf("status after panic: %v", err)
	}
	if got.ID != ideaID {
		t.Errorf("idea lost after panic: got ID %q, want %q", got.ID, ideaID)
	}

	// Transaction that returns error should also roll back cleanly
	err = db.Transaction(func(tx *store.DB) error {
		return store.ErrNotFound
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// DB still usable
	got2, _, err := ebp.Status(ctx, ideaID)
	if err != nil {
		t.Fatalf("status after error txn: %v", err)
	}
	if got2.ID != ideaID {
		t.Errorf("idea lost after error txn: got ID %q, want %q", got2.ID, ideaID)
	}
}

func TestMultiConnectionConcurrentPromoteAndAddDebt(t *testing.T) {
	tmpFile := t.TempDir() + "/test_multi.db"

	// Open two separate connections to the same file-backed WAL database
	db1, err := store.Open(tmpFile)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}
	defer db1.Close()

	db2, err := store.Open(tmpFile)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()

	ebp1 := New(db1)
	ebp2 := New(db2)
	ctx := context.Background()

	// Create idea on connection 1
	idea, err := ebp1.Capture(ctx, "Alice", "multi-conn test", "", "alice")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	ideaID := idea.ID

	// Retire all debt on connection 2 — validates cross-connection shared state
	for _, item := range domain.AllDebtItems {
		if err := ebp2.Retire(ctx, ideaID, item, "retired", "alice"); err != nil {
			t.Fatalf("retire %s: %v", item, err)
		}
	}

	const iterations = 50
	for i := 0; i < iterations; i++ {
		// Reset: add debt on conn1 to un-promote
		if err := ebp1.AddDebt(ctx, ideaID, domain.DebtNeedObstruction, "reset"); err != nil {
			t.Fatalf("iteration %d reset: %v", i, err)
		}

		// Concurrent Promote on conn1 + AddDebt on conn2
		type result struct{ err error }
		done := make(chan result, 2)

		go func() { done <- result{ebp1.Promote(ctx, ideaID, "promoter")} }()
		go func() { done <- result{ebp2.AddDebt(ctx, ideaID, domain.DebtNeedObstruction, "debter")} }()

		r1, r2 := <-done, <-done

		// SQLITE_BUSY is expected: one transaction must lose the write conflict.
		// Both failing is acceptable. Neither must silently corrupt state.
		if r1.err != nil && r2.err != nil {
			// Both failed — write conflict resolved correctly by WAL rollback
		}

		// Core invariant: never promoted with open debt
		got, debt, err := ebp1.Status(ctx, ideaID)
		if err != nil {
			t.Fatalf("iteration %d status: %v", i, err)
		}
		openDebt := 0
		for _, d := range debt {
			if !d.Retired {
				openDebt++
			}
		}
		if got.Promoted && openDebt > 0 {
			t.Fatalf("iteration %d: promoted with %d open debt items", i, openDebt)
		}
	}
}
