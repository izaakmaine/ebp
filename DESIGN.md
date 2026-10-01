# EBP 2.1 Design Document

## What EBP Is

EBP is an epistemic ledger/grammar implementation of the Elephant Bridge Protocol v2.1.

It is a notebook with a conscience: it preserves imagination (ideas enter free) while preventing unearned promotion (debt must be retired before promotion).

## Architectural Boundaries

```
Agent / Runtime     = agency
EBP                 = epistemic state
Conductor           = coordination
Solvent             = consequential authority
Executor            = external effect
External SOR        = effect outcome
```

Core invariant:

```
CAPABILITY ≠ WORK ≠ AUTHORITY ≠ EXECUTION
```

## Ownership

| Component | Owns |
|-----------|------|
| EBP | Ideas, debt, debt retirement, promotion |
| Conductor | Tasks, claims, dependencies, coordination history |
| Solvent | Consequential authorization |
| Executor | External effects |
| Pizza Bot | Agent context, HITL presentation |

## What EBP Is NOT

- **Not a workflow engine** — no DAG execution, conditional branching, or parallel fork/join
- **Not a project manager** — no task assignment, scheduling, or resource allocation
- **Not a scheduler** — no background workers, cron, or timed execution
- **Not an authorization system** — no RBAC, capability matching, or policy evaluation
- **Not an execution engine** — no work execution, effect brokering, or consequence management
- **Not a policy engine** — no rule evaluation or constraint solving
- **Not a domain reasoning engine** — no physics, mathematics, or domain-specific inference
- **Not a cascade/orchestration engine** — cross-idea dependency behavior is explicitly deferred

## Independence

- EBP is independently replaceable. It can be swapped for another epistemic implementation without affecting Conductor or Solvent.
- Pizza Bot is not an EBP dependency. EBP can be used without Pizza Bot.
- BM-IST is not encoded into the core. Domain-specific content belongs in a replaceable skill layer.
- No Conductor or Solvent changes are required for EBP to function.

## Persistence

SQLite via pure Go driver (`modernc.org/sqlite`). WAL mode. Foreign keys enabled.

Three tables:
- `ebp_idea` — idea records
- `ebp_debt_item` — debt obligations per idea
- `ebp_event` — append-only audit log

No distributed database. No event bus. No external infrastructure requirement.

## Integration Boundary

EBP exposes a service/library interface with seven operations.

Integration with Conductor and Solvent is deferred. When integrated:
- Pizza Bot agent calls EBP for epistemic state
- Pizza Bot agent calls Conductor for coordination
- Solvent handles consequential authorization separately
- EBP never authorizes or executes consequential actions

## Adapter Layers

EBP core is exposed through two thin adapters:

- **CLI** (`cmd/ebp/main.go`) — human-facing command line
- **MCP** (`cmd/ebp-mcp/main.go`) — agent-facing MCP stdio server

Both adapters call the same `service.EBP` methods. Neither adds business
logic or persistence. The adapters are independently replaceable.

The MCP adapter uses `github.com/mark3labs/mcp-go` and is confined to
`internal/mcp/` and `cmd/ebp-mcp/`. The EBP core (`internal/service/`,
`internal/store/`, `internal/domain/`) has zero MCP imports.

## Promotion Rule

An idea is promoted when:
1. All debt items are retired (open debt count = 0)
2. `containsFinalTruthClaim` is false

Promotion is a recorded event, not just a derived state. This gives promotion a durable, auditable fact.

New debt after promotion un-promotes the idea. Promotion is not irreversible truth.

## Debt Semantics

- Debt does not kill. Unpaid ideas remain alive.
- Debt is forever payable. No expiration date.
- One move at a time. Each retirement retires exactly one debt item.
- Evidence is preserved for auditability.
- New evidence creates new debt, even on promoted ideas.
