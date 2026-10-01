package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/PithomLabs/ebp/internal/service"
)

// Server wraps an EBP service and exposes it through MCP tools.
type Server struct {
	ebp *service.EBP
	s   *server.MCPServer
}

// NewServer creates an MCP server backed by the given EBP service.
func NewServer(ebp *service.EBP) *Server {
	s := &Server{ebp: ebp}
	s.s = server.NewMCPServer(
		"ebp-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)
	s.registerTools()
	return s
}

// ServeStdio starts the MCP server on stdio transport.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.s)
}

func (s *Server) registerTools() {
	s.s.AddTool(
		mcp.NewTool("ebp_capture",
			mcp.WithDescription("Capture a new idea with full starting debt. Minimum required: owner + claim."),
			mcp.WithString("owner", mcp.Required(), mcp.Description("Idea owner")),
			mcp.WithString("claim", mcp.Required(), mcp.Description("Idea claim")),
			mcp.WithString("source", mcp.Description("Optional source reference")),
		),
		s.handleCapture,
	)

	s.s.AddTool(
		mcp.NewTool("ebp_status",
			mcp.WithDescription("Read current idea state and debt items from a consistent snapshot."),
			mcp.WithString("idea_id", mcp.Required(), mcp.Description("Idea ID")),
		),
		s.handleStatus,
	)

	s.s.AddTool(
		mcp.NewTool("ebp_retire",
			mcp.WithDescription("Retire exactly one debt item with evidence."),
			mcp.WithString("idea_id", mcp.Required(), mcp.Description("Idea ID")),
			mcp.WithString("item", mcp.Required(), mcp.Description("Debt item name")),
			mcp.WithString("evidence", mcp.Required(), mcp.Description("Evidence for retirement")),
		),
		s.handleRetire,
	)

	s.s.AddTool(
		mcp.NewTool("ebp_add_debt",
			mcp.WithDescription("Add one canonical debt item to an idea. If promoted, new debt un-promotes it."),
			mcp.WithString("idea_id", mcp.Required(), mcp.Description("Idea ID")),
			mcp.WithString("item", mcp.Required(), mcp.Description("Debt item name")),
		),
		s.handleAddDebt,
	)

	s.s.AddTool(
		mcp.NewTool("ebp_promote",
			mcp.WithDescription("Promote idea. Requires empty debt, no final-truth claim, and not already promoted."),
			mcp.WithString("idea_id", mcp.Required(), mcp.Description("Idea ID")),
		),
		s.handlePromote,
	)

	s.s.AddTool(
		mcp.NewTool("ebp_mark_final_truth_claim",
			mcp.WithDescription("Flag final-truth language on an idea. Blocks promotion."),
			mcp.WithString("idea_id", mcp.Required(), mcp.Description("Idea ID")),
		),
		s.handleMarkFinalTruthClaim,
	)

	s.s.AddTool(
		mcp.NewTool("ebp_clear_final_truth_claim",
			mcp.WithDescription("Clear final-truth flag on an idea. Allows promotion."),
			mcp.WithString("idea_id", mcp.Required(), mcp.Description("Idea ID")),
		),
		s.handleClearFinalTruthClaim,
	)
}

// CallTool is a test helper that invokes a tool handler directly.
func (s *Server) CallTool(toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}
	return s.handleTool(toolName, req)
}

func (s *Server) handleTool(name string, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	switch name {
	case "ebp_capture":
		return s.handleCapture(context.Background(), req)
	case "ebp_status":
		return s.handleStatus(context.Background(), req)
	case "ebp_retire":
		return s.handleRetire(context.Background(), req)
	case "ebp_add_debt":
		return s.handleAddDebt(context.Background(), req)
	case "ebp_promote":
		return s.handlePromote(context.Background(), req)
	case "ebp_mark_final_truth_claim":
		return s.handleMarkFinalTruthClaim(context.Background(), req)
	case "ebp_clear_final_truth_claim":
		return s.handleClearFinalTruthClaim(context.Background(), req)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unknown tool: %s", name)), nil
	}
}

func (s *Server) handleCapture(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	owner, _ := args["owner"].(string)
	claim, _ := args["claim"].(string)
	source, _ := args["source"].(string)

	idea, err := s.ebp.Capture(ctx, owner, claim, source, "mcp")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, _ := json.Marshal(idea)
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	ideaID, _ := args["idea_id"].(string)

	idea, debt, err := s.ebp.Status(ctx, ideaID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result := map[string]interface{}{
		"idea": idea,
		"debt": debt,
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleRetire(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	ideaID, _ := args["idea_id"].(string)
	item, _ := args["item"].(string)
	evidence, _ := args["evidence"].(string)

	if err := s.ebp.Retire(ctx, ideaID, item, evidence, "mcp"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Debt retired: %s", item)), nil
}

func (s *Server) handleAddDebt(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	ideaID, _ := args["idea_id"].(string)
	item, _ := args["item"].(string)

	if err := s.ebp.AddDebt(ctx, ideaID, item, "mcp"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Debt added: %s", item)), nil
}

func (s *Server) handlePromote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	ideaID, _ := args["idea_id"].(string)

	if err := s.ebp.Promote(ctx, ideaID, "mcp"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Idea promoted."), nil
}

func (s *Server) handleMarkFinalTruthClaim(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	ideaID, _ := args["idea_id"].(string)

	if err := s.ebp.MarkFinalTruthClaim(ctx, ideaID, "mcp"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Final-truth claim flagged."), nil
}

func (s *Server) handleClearFinalTruthClaim(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	ideaID, _ := args["idea_id"].(string)

	if err := s.ebp.ClearFinalTruthClaim(ctx, ideaID, "mcp"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Final-truth claim cleared."), nil
}
