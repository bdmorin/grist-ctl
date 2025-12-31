# Beads Agent Integration Research

**Date**: 2025-12-31
**Scope**: Investigation into beads automatic agent interaction tracking
**Question**: Does beads automatically capture rich agent work metadata?

## Executive Summary

**Answer: No.** Beads provides infrastructure for rich agent tracking but does NOT automatically populate it. The system requires explicit instrumentation or manual workflow discipline.

## What Beads Auto-Tracks (Out of Box)

✅ **Issue State Management** (automatic via daemon):
- Issue CRUD operations → `.beads/issues.jsonl`
- Status transitions, priority changes, label updates
- Dependencies, assignees, timestamps
- Actor attribution (who made changes)
- Git integration (auto-export with 5s debounce)

❌ **NOT Auto-Tracked** (requires manual/custom instrumentation):
- LLM prompts and responses
- Tool execution details
- Agent reasoning/decision logs
- Work narrative ("tried X, then Y")
- Error details and stack traces

## The Empty `interactions.jsonl` Mystery - Solved

**File**: `.beads/interactions.jsonl`
**Status**: Empty (0 lines) in active beads projects
**Reason**: This is **expected behavior**

The audit trail is a **manual API**, not automatic logging:
```bash
bd audit record --kind=llm_call --issue-id=bd-123 --prompt="..." --response="..."
bd audit record --kind=tool_call --issue-id=bd-123 --tool-name="go test"
```

This file only gets populated when:
1. Custom agent wrappers explicitly call `bd audit record`
2. Integration layers instrument LLM/tool interactions
3. Developers manually log key decisions

## Beads Architecture: What's Provided

### 1. Daemon (Background Sync)
```bash
bd daemon --status
# Running: Auto-exports DB → JSONL
# Optional: --auto-commit, --auto-push, --auto-pull
```

### 2. Git Hooks (File System Sync)
```bash
bd hooks install
# Installs: pre-commit, post-merge, pre-push, post-checkout
# Purpose: Keep DB and JSONL in sync across git operations
```

### 3. Claude Code Hooks (Context Injection)
```bash
bd setup claude --project
# Creates .claude/settings.local.json with:
# - SessionStart: bd prime (inject workflow context)
# - PreCompact: bd prime (refresh before compression)
```

### 4. Manual Richness APIs
```bash
bd comments add <id> "Work narrative here"
bd close <id> --reason="Detailed completion notes"
bd audit record --kind=llm_call ...
```

## Gap Analysis: Expected vs. Actual

| Feature | Expected | Actual | Gap |
|---------|----------|--------|-----|
| Issue tracking | ✅ Auto | ✅ Auto | None |
| Work comments | ✅ Auto? | ❌ Manual | **Large** |
| Agent audit logs | ✅ Auto? | ❌ Manual | **Large** |
| Close reasons | ✅ Auto? | ❌ Manual | **Medium** |
| Tool call logs | ✅ Auto? | ❌ Manual | **Large** |

## Beads Design Philosophy

From documentation review:
- **Auto-track WHAT changed** (issue state, deps, labels)
- **Manually track WHY it changed** (comments, rationale, audit)
- **Provide APIs** for custom integrations
- **Not a turnkey solution** for agent observability

The `interactions.jsonl` is **infrastructure**, not a feature.

## What Would Full Integration Require?

To achieve automatic rich metadata, you would need:

### Option 1: Claude Code Extension (Ideal)
Build a Claude Code hook/plugin that:
```typescript
// Pseudocode
onToolCall((tool, args, result) => {
  exec(`bd audit record --kind=tool_call --tool-name=${tool} --issue-id=${currentIssue}`)
})

onLLMCall((prompt, response) => {
  exec(`bd audit record --kind=llm_call --prompt="${prompt}" --response="${response}"`)
})
```

**Scope**: Medium complexity, requires Claude SDK knowledge
**Benefit**: Works globally across all repos

### Option 2: Wrapper Script (Pragmatic)
Create `claude-with-beads` wrapper:
```bash
#!/bin/bash
# Intercept Claude calls, log to beads
claude "$@" 2>&1 | tee >(parse-and-log-to-beads)
```

**Scope**: Low complexity, brittle
**Benefit**: Quick prototype

### Option 3: Enhanced Manual Protocol (Immediate)
Discipline-based approach:
```bash
# Start work
bd update <id> --status=in_progress
bd comments add <id> "Starting: plan is X, Y, Z"

# During work
bd comments add <id> "Progress: completed A, discovered issue B"

# Complete work
bd comments add <id> "Files changed: handlers.go, tests.go | Approach: validation pattern | Tests: 12 new cases passing"
bd close <id> --reason="Added input validation. All tests pass. Ready for review."
```

**Scope**: Zero development, requires discipline
**Benefit**: Works today

## Recommendations

### For Cross-Repo Beads Setup

**Global Configuration** (do once):
```bash
# Install beads globally (already done)
mise use -g go:github.com/steveyegge/beads/cmd/bd@latest

# Configure Claude Code globally
bd setup claude  # Not --project

# Configure git hooks template
git config --global init.templatedir ~/.git-templates
bd hooks install --global  # If this exists
```

**Per-Repo** (minimal):
```bash
cd /path/to/repo
bd init
# Hooks auto-install from template
# Claude hooks already global
```

### For Agent Metadata Capture

**Short Term** (this week):
- Use enhanced manual protocol (Option 3)
- Update session close checklist to require comments
- Train agents to use `bd comments add` frequently

**Medium Term** (this month):
- Build simple wrapper script (Option 2)
- Test with one repo
- Iterate based on usage

**Long Term** (if valuable):
- Build Claude Code extension (Option 1)
- Contribute back to beads project
- Share with community

## Where to Do This Work?

### ❌ NOT in gristle repo
- Out of scope for gristle project
- Infrastructure concern, not app concern
- Would clutter gristle with beads-specific code

### ✅ In beads repo
**Path**: `/Users/bdmorin/src/github.com/steveyegge/beads`

**Reasons**:
1. Upstream contribution opportunity
2. Proper context for beads development
3. Could benefit beads community
4. Might already have related work/issues

**First Steps**:
```bash
cd /Users/bdmorin/src/github.com/steveyegge/beads
bd list --label=claude --label=agent --label=integration
bd list --label=audit
git log --all --grep="audit" --grep="agent" --oneline
```

### ✅ In separate dotfiles/tools repo
**Alternative**: Create `~/src/beads-claude-integration`

**Reasons**:
1. Personal tool, not upstream contribution (yet)
2. Experimental/prototype phase
3. Can iterate quickly
4. Easy to share as gist/repo later

## Next Steps

1. **Decide venue**: beads repo vs. standalone repo
2. **Check beads issues**: See if this work already exists/planned
3. **Start small**: Enhanced manual protocol + session checklist
4. **Measure value**: Does rich metadata actually help? Track for 2 weeks
5. **Iterate**: If valuable, build wrapper; if very valuable, build extension

## References

- Beads source: `/Users/bdmorin/src/github.com/steveyegge/beads`
- Beads docs: `/Users/bdmorin/src/github.com/steveyegge/beads/docs`
- This research: `grist-ctl/docs/research/BEADS_AGENT_INTEGRATION_RESEARCH.md`
- Related: Session close protocol in startup hook

## Conclusion

**You didn't misconfigure anything.** Beads simply doesn't have automatic agent metadata capture - it provides infrastructure that requires integration work. The path forward depends on how much value you place on rich work history vs. the effort to instrument it.

Recommendation: Start with enhanced manual protocol, measure value, then decide if automation is worth building.
