package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/PithomLabs/ebp/internal/domain"
	"github.com/PithomLabs/ebp/internal/service"
	"github.com/PithomLabs/ebp/internal/store"
)

func setupTest(t *testing.T) *Server {
	t.Helper()
	db, err := store.OpenTest()
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ebp := service.New(db)
	return NewServer(ebp)
}

func resultText(result *mcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestToolRegistration(t *testing.T) {
	s := setupTest(t)

	tools := []string{
		"ebp_capture",
		"ebp_status",
		"ebp_retire",
		"ebp_add_debt",
		"ebp_promote",
		"ebp_mark_final_truth_claim",
		"ebp_clear_final_truth_claim",
	}

	for _, tool := range tools {
		result, err := s.CallTool(tool, map[string]interface{}{})
		if err != nil {
			t.Fatalf("tool %s returned transport error: %v", tool, err)
		}
		if result.IsError {
			text := resultText(result)
			if strings.Contains(text, "unknown tool") {
				t.Errorf("tool %s not registered", tool)
			}
		}
	}

	// Unknown tool should fail
	result, err := s.CallTool("nonexistent_tool", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unknown tool returned transport error: %v", err)
	}
	if !result.IsError {
		t.Error("unknown tool should return error result")
	}
}

func TestMCPCapture(t *testing.T) {
	s := setupTest(t)

	result, err := s.CallTool("ebp_capture", map[string]interface{}{
		"owner":  "Alice",
		"claim":  "test claim",
		"source": "test source",
	})
	if err != nil {
		t.Fatalf("capture error: %v", err)
	}
	if result.IsError {
		t.Fatalf("capture failed: %v", resultText(result))
	}

	var idea domain.Idea
	if err := json.Unmarshal([]byte(resultText(result)), &idea); err != nil {
		t.Fatalf("failed to parse idea: %v", err)
	}
	if idea.Owner != "Alice" {
		t.Errorf("owner = %q, want %q", idea.Owner, "Alice")
	}
	if idea.Claim != "test claim" {
		t.Errorf("claim = %q, want %q", idea.Claim, "test claim")
	}
}

func TestMCPStatus(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	result, err := s.CallTool("ebp_status", map[string]interface{}{
		"idea_id": idea.ID,
	})
	if err != nil {
		t.Fatalf("status error: %v", err)
	}
	if result.IsError {
		t.Fatalf("status failed: %v", resultText(result))
	}

	var status struct {
		Idea domain.Idea       `json:"idea"`
		Debt []domain.DebtItem `json:"debt"`
	}
	if err := json.Unmarshal([]byte(resultText(result)), &status); err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}
	if status.Idea.ID != idea.ID {
		t.Errorf("status idea ID = %q, want %q", status.Idea.ID, idea.ID)
	}
	if len(status.Debt) != 6 {
		t.Errorf("debt count = %d, want 6", len(status.Debt))
	}
}

func TestMCPAddDebt(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	result, err := s.CallTool("ebp_add_debt", map[string]interface{}{
		"idea_id": idea.ID,
		"item":    domain.DebtNeedObstruction,
	})
	if err != nil {
		t.Fatalf("add debt error: %v", err)
	}
	if result.IsError {
		t.Fatalf("add debt failed: %v", resultText(result))
	}
	if !strings.Contains(resultText(result), "needObstruction") {
		t.Errorf("unexpected result: %s", resultText(result))
	}
}

func TestMCPRetire(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	result, err := s.CallTool("ebp_retire", map[string]interface{}{
		"idea_id":  idea.ID,
		"item":     domain.DebtNeedMap,
		"evidence": "map defined",
	})
	if err != nil {
		t.Fatalf("retire error: %v", err)
	}
	if result.IsError {
		t.Fatalf("retire failed: %v", resultText(result))
	}
	if !strings.Contains(resultText(result), "needMap") {
		t.Errorf("unexpected result: %s", resultText(result))
	}
}

func TestMCPPromote(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	for _, item := range domain.AllDebtItems {
		s.CallTool("ebp_retire", map[string]interface{}{
			"idea_id":  idea.ID,
			"item":     item,
			"evidence": "retired",
		})
	}

	result, err := s.CallTool("ebp_promote", map[string]interface{}{
		"idea_id": idea.ID,
	})
	if err != nil {
		t.Fatalf("promote error: %v", err)
	}
	if result.IsError {
		t.Fatalf("promote failed: %v", resultText(result))
	}
	if resultText(result) != "Idea promoted." {
		t.Errorf("unexpected result: %s", resultText(result))
	}
}

func TestMCPMarkFinalTruthClaim(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	result, err := s.CallTool("ebp_mark_final_truth_claim", map[string]interface{}{
		"idea_id": idea.ID,
	})
	if err != nil {
		t.Fatalf("mark error: %v", err)
	}
	if result.IsError {
		t.Fatalf("mark failed: %v", resultText(result))
	}
	if resultText(result) != "Final-truth claim flagged." {
		t.Errorf("unexpected result: %s", resultText(result))
	}
}

func TestMCPClearFinalTruthClaim(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	s.CallTool("ebp_mark_final_truth_claim", map[string]interface{}{
		"idea_id": idea.ID,
	})

	result, err := s.CallTool("ebp_clear_final_truth_claim", map[string]interface{}{
		"idea_id": idea.ID,
	})
	if err != nil {
		t.Fatalf("clear error: %v", err)
	}
	if result.IsError {
		t.Fatalf("clear failed: %v", resultText(result))
	}
	if resultText(result) != "Final-truth claim cleared." {
		t.Errorf("unexpected result: %s", resultText(result))
	}
}

func TestMCPPromoteBlockedWithDebt(t *testing.T) {
	s := setupTest(t)

	capResult, _ := s.CallTool("ebp_capture", map[string]interface{}{
		"owner": "Alice", "claim": "test",
	})
	var idea domain.Idea
	json.Unmarshal([]byte(resultText(capResult)), &idea)

	result, err := s.CallTool("ebp_promote", map[string]interface{}{
		"idea_id": idea.ID,
	})
	if err != nil {
		t.Fatalf("promote returned transport error: %v", err)
	}
	if !result.IsError {
		t.Error("promote with debt should return error result")
	}
}

func TestMCPInvalidArguments(t *testing.T) {
	s := setupTest(t)

	result, err := s.CallTool("ebp_capture", map[string]interface{}{
		"claim": "test",
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !result.IsError {
		t.Error("missing owner should return error result")
	}

	result, err = s.CallTool("ebp_status", map[string]interface{}{})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !result.IsError {
		t.Error("missing idea_id should return error result")
	}
}
