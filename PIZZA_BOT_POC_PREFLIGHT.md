# PIZZA_BOT_POC_PREFLIGHT.md

**Date:** 2026-09-13
**Purpose:** Final preflight verification for BM-IST Synthesis POC
**Status:** READ-ONLY — no code modified

---

## 1. ENVIRONMENT USED

| Item | Value |
|------|-------|
| Host | Linux (x86_64) |
| Go | /usr/local/go/bin/go |
| Pizza Bot | pizza-bot-oss 1.0.0 (installed via dpkg) |
| CockroachDB | cockroach binary present at /home/chaschel/.local/bin/cockroach |
| SQLite | /usr/bin/sqlite3 |
| BM-IST seed | /home/chaschel/Documents/go/bm-ist/BMIST_POC_SEED.md |

---

## 2. BINARY CHECKS

| Binary | Status | Path |
|--------|--------|------|
| ebp-mcp | PRESENT | /home/chaschel/Documents/go/ebp/bin/ebp-mcp |
| conductor | PRESENT | /home/chaschel/Documents/go/conductor/bin/conductor |
| conductor-mcp-bridge | PRESENT | /home/chaschel/Documents/go/conductor-mcp-bridge/bin/conductor-mcp-bridge |
| integration-tools | PRESENT | /home/chaschel/Documents/go/integration-tools/bin/integration-tools |
| solvent-mcp | PRESENT | /home/chaschel/Documents/go/solvent-main/bin/solvent-mcp |

All required binaries exist and are valid ELF x86-64 executables.

---

## 3. ENVIRONMENT CHECKS

### Required Variables

| Variable | Status |
|----------|--------|
| EBP_MCP_PATH | NOT SET |
| EBP_DB_PATH | NOT SET |
| CONDUCTOR_BRIDGE_PATH | NOT SET |
| CONDUCTOR_URL | NOT SET |
| CONDUCTOR_API_KEY | NOT SET |
| CONDUCTOR_MCP_AGENT_ID | NOT SET |
| INTEGRATION_TOOLS_PATH | NOT SET |
| FABLE_DSN | NOT SET |
| SOLVENT_FIXTURE_ROOT | NOT SET |

**Finding:** No integration environment variables are set. The `.env` file at `/home/chaschel/Documents/.env` exists but is empty (0 bytes).

**Impact:** Pizza Bot cannot resolve the `${VARIABLE}` placeholders in `~/.pizza-bot-oss/.mcp.json`. All MCP server commands will fail to launch.

---

## 4. SERVICE CHECKS

| Service | Status |
|---------|--------|
| EBP MCP | NOT RUNNING |
| Conductor API | NOT RUNNING |
| CockroachDB | NOT RUNNING |
| Solvent MCP | NOT RUNNING |

**Finding:** No integration services are currently running. A process scan found zero EBP/Conductor/Solvent/Pizza Bot processes.

---

## 5. PIZZA BOT MCP CONFIGURATION

**File:** `~/.pizza-bot-oss/.mcp.json`

**Findings:**
- All four servers are configured with unresolved template variables (`${EBP_MCP_PATH}`, etc.)
- Solvent is explicitly disabled (`"enabled": false`)
- This configuration will NOT work until template variables are resolved

---

## 6. MCP DISCOVERY

Could not be verified — no MCP servers are running and config uses unresolved template variables.

**Expected from source:**
- EBP: 7 tools
- Conductor bridge: source present
- integration-tools: source present
- Solvent: source present

---

## 7. SKILL READINESS

| Skill | Status |
|-------|--------|
| ebp-conductor | NOT FOUND |
| solvent-governance | NOT FOUND |

**Finding:** Neither Skill was found on the filesystem.

---

## 8. ARCHITECTURAL BOUNDARY VERIFICATION

| Component | Status |
|-----------|--------|
| EBP unchanged | YES |
| Conductor unchanged | YES |
| Solvent unchanged | YES |
| Pizza Bot runtime unchanged | YES |
| No domain-specific infrastructure | YES |
| No workflow engine | YES |
| No scheduler | YES |
| No EBP cascade | YES |
| No new persistence layer | YES |

---

## 9. BM-IST SEED READINESS

**File:** `/home/chaschel/Documents/go/bm-ist/BMIST_POC_SEED.md`
**Status:** PRESENT (5,757 bytes)

---

## BLOCKERS

1. **Environment variables not set** — All 9 required integration variables are unset
2. **No services running** — EBP MCP, Conductor API, CockroachDB, Solvent MCP all stopped
3. **Pizza Bot MCP config unresolved** — `~/.pizza-bot-oss/.mcp.json` contains unresolved `${...}` placeholders
4. **Skills missing** — `ebp-conductor` and `solvent-governance` Skills not installed
5. **Solvent disabled** — Explicitly disabled in MCP config

---

## FINAL VERDICT

**BLOCKED — Pizza Bot preflight failed.**

The BM-IST Synthesis POC cannot proceed until:
1. All 9 environment variables are set with concrete values
2. `~/.pizza-bot-oss/.mcp.json` template variables are resolved
3. `ebp-conductor` and `solvent-governance` Skills are installed and READY
4. EBP MCP, Conductor API, and optionally Solvent MCP services are started
5. CockroachDB is running if Solvent consequential-path verification is required

**Code was NOT modified.**
