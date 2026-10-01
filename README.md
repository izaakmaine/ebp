# EBP 2.1 — Elephant Bridge Protocol

A standalone epistemic ledger implementing the Elephant Bridge Protocol v2.1.

**Ideas enter free. Promotion costs debt.**

EBP tracks ideas, their outstanding epistemic debt, and promotion readiness. It is a notebook with a conscience: wide open to imagination, ruthless against unearned promotion.

## Core Doctrine

```
Ideas enter free.
Promotion costs debt.
Debt does not kill.
Debt is forever payable.
New evidence creates new debt.
No final-truth claim may be promoted.
Accounting must never become the work.
```

## The Seven Operations

| Command | Purpose |
|---------|---------|
| `capture` | Create a new idea with full starting debt |
| `status` | Read current idea state and debt |
| `retire` | Retire exactly one debt item with evidence |
| `add-debt` | Add one canonical debt item |
| `promote` | Promote idea (requires empty debt + no final-truth claim) |
| `mark-final-truth-claim` | Flag final-truth language (blocks promotion) |
| `clear-final-truth-claim` | Clear final-truth flag (allows promotion) |

## The Six Canonical Debt Items

- `needMap` — idea claims a relation between structures
- `needInvariant` — idea claims something survives translation
- `needToyCheck` — idea has map + invariant but no finite test
- `needNullModel` — idea claims uniqueness or superiority
- `needObstruction` — known blockers or no-go results apply
- `needFaithfulnessReview` — does the formalization match the claim?

## Building

```bash
# CLI
go build -o bin/ebp ./cmd/ebp

# MCP server
go build -o bin/ebp-mcp ./cmd/ebp-mcp
```

## Usage

```bash
# Capture an idea
./bin/ebp capture "Alice" "Boundary capacity may unify area and entropy"

# Check status
./bin/ebp status <idea-id>

# Retire debt
./bin/ebp retire <idea-id> needInvariant "Invariant: distinguishability capacity I(B)"

# Add new debt
./bin/ebp add-debt <idea-id> needObstruction

# Promote (when ready)
./bin/ebp promote <idea-id>

# Flag/clear final-truth language
./bin/ebp mark-final-truth-claim <idea-id>
./bin/ebp clear-final-truth-claim <idea-id>
```

## MCP Server

EBP exposes an MCP server for integration with MCP-compatible agents (e.g., Pizza Bot).

### Running

```bash
go build -o bin/ebp-mcp ./cmd/ebp-mcp
./bin/ebp-mcp --db ebp.db
```

### Pizza Bot Integration

Add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "ebp": {
      "command": "/path/to/ebp-mcp",
      "args": ["--db", "/path/to/ebp.db"]
    }
  }
}
```

### Exposed Tools

| Tool | Description |
|------|-------------|
| `ebp_capture` | Capture a new idea with full starting debt |
| `ebp_status` | Read current idea state and debt items |
| `ebp_retire` | Retire exactly one debt item with evidence |
| `ebp_add_debt` | Add one canonical debt item |
| `ebp_promote` | Promote idea (requires empty debt + no final-truth claim) |
| `ebp_mark_final_truth_claim` | Flag final-truth language (blocks promotion) |
| `ebp_clear_final_truth_claim` | Clear final-truth flag (allows promotion) |

The MCP server is an adapter over the frozen EBP 2.1 core. It does not
duplicate business logic or add new persistence.

## Testing

```bash
go test ./...
```

## Database

Default: `ebp.db` in current directory. Override with `--db <path>`.

SQLite via pure Go driver. WAL mode. Foreign keys enabled. Survives process restart.

## Architecture

EBP is a standalone, independently replaceable epistemic component.

- **EBP** owns epistemic state (ideas, debt, promotion)
- **Conductor** owns coordination (tasks, claims, dependencies)
- **Solvent** owns consequential authorization
- **Executor** owns external effects

EBP does NOT become a workflow engine, project manager, scheduler, authorization system, or execution engine.

### Adapter Layers

EBP core is exposed through two thin adapters:

- **CLI** (`cmd/ebp/main.go`) — human-facing command line
- **MCP** (`cmd/ebp-mcp/main.go`) — agent-facing MCP stdio server

Both adapters call the same `service.EBP` methods. Neither adds business
logic or persistence. The adapters are independently replaceable.


## License

MIT
