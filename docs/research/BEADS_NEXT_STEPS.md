# Beads Agent Integration: Where to Go From Here

**Date**: 2025-12-31
**Context**: [BEADS_AGENT_INTEGRATION_RESEARCH.md](./BEADS_AGENT_INTEGRATION_RESEARCH.md)

## Assessment: Where Should This Work Happen?

### ✅ RECOMMENDED: Work in Beads Repo

**Location**: `/Users/bdmorin/src/github.com/steveyegge/beads`

**Evidence**:
1. **Beads dogfoods itself**: Has `.beads/` directory with 811KB of issues
2. **Active agent work**: Recent commits show agent-related features:
   - `feat: add prepare-commit-msg hook for agent identity trailers (bd-luso)`
   - `feat: add structured labels for agent beads (bd-g7eq)`
   - `feat: add --town flag to bd activity for aggregated cross-rig feed (bd-dx6e)`
3. **Empty interactions.jsonl**: Even beads project has 0 bytes - confirms this is expected
4. **Upstream opportunity**: Your work could benefit beads community
5. **Proper context**: Beads maintainers are actively working on agent features

### ❌ NOT RECOMMENDED: Gristle Repo

**Reasons**:
- Out of scope for gristle (tool integration, not app feature)
- Would clutter gristle with infrastructure code
- Won't benefit other beads users

### 🤔 ALTERNATIVE: Standalone Tool Repo

**If upstream contribution isn't desired**:
- Create `~/src/beads-claude-bridge` or similar
- Prototype independently
- Share as separate tool
- Later: contribute upstream if valuable

## Immediate Action Plan

### Phase 1: Research in Beads Repo (Today)

```bash
cd /Users/bdmorin/src/github.com/steveyegge/beads

# Check existing work
bd list --label=agent --label=claude --label=audit
bd search "automatic.*audit"
bd search "claude.*hook"

# Look for related issues
bd list --status=open | grep -i "agent\|audit\|claude"

# Check git history
git log --all --grep="audit" --grep="interactions.jsonl" --oneline
```

**Goal**: Understand if this work already exists or is planned

### Phase 2: Upstream First (This Week)

**Option A: Enhancement to Existing**
If beads has related work-in-progress:
- Comment on existing issue
- Offer to help implement
- Collaborate with beads maintainers

**Option B: New Feature Proposal**
If no existing work:
- Create beads issue: "Automatic agent interaction logging for Claude Code"
- Propose design:
  - Hook into Claude Code tool/LLM calls
  - Auto-populate `interactions.jsonl`
  - Optional: Make generic for any agent framework
- Get feedback before building

### Phase 3: Implementation (This Month)

**Start Small**: Enhanced manual protocol (no code)
```bash
# Update session close checklist
# Train to use: bd comments add <id> "narrative"
# Measure: Does this actually help?
```

**If Valuable**: Build automation
```bash
# Create: ~/.local/bin/bd-claude-hook
# Instrument: Claude Code tool calls → bd audit record
# Test: In one repo for 2 weeks
# Iterate: Based on usage patterns
```

**If Very Valuable**: Contribute upstream
```bash
# Polish implementation
# Add tests
# Documentation
# PR to beads repo
```

## Global Beads + Claude Setup (Once Per Machine)

### Current State ✅
```bash
# Already done:
mise use -g go:github.com/steveyegge/beads/cmd/bd@latest
bd setup claude  # (was --project, but should be global)
```

### Recommended Global Config

**File**: `~/.config/beads/config.yaml` (if beads supports this)
```yaml
# Default actor for all repos
actor: "bdmorin"

# Auto-start daemon
auto-start-daemon: true

# Flush debounce
flush-debounce: "5s"
```

**Per-Repo** (minimal setup):
```bash
cd /path/to/new/repo
bd init
# That's it - hooks auto-configure from global
```

### Git Hooks Template (Global)

If beads supports:
```bash
# One-time setup
git config --global init.templatedir ~/.git-templates
mkdir -p ~/.git-templates/hooks
bd hooks install --template ~/.git-templates/hooks

# All new repos auto-get hooks
git init /path/to/new/repo  # hooks auto-installed
```

**Current Gap**: Beads may not support `--template` flag yet
- Check: `bd hooks install --help`
- If not: This could be another upstream contribution

## Decision Tree

```
Do you want rich agent metadata?
├─ No → Stop here, current setup is fine
└─ Yes → How much value?
    ├─ Low → Use enhanced manual protocol (comments + close reasons)
    ├─ Medium → Build personal wrapper script
    └─ High → Contribute to beads upstream
        ├─ Check beads repo for existing work
        ├─ Create issue / comment on existing
        ├─ Prototype in beads repo
        └─ Submit PR if valuable
```

## Next Session: Move to Beads Context

**When continuing this work**:

1. **Switch repos**:
   ```bash
   cd /Users/bdmorin/src/github.com/steveyegge/beads
   ```

2. **Research first**:
   ```bash
   bd list --label=agent
   bd search "audit.*automatic"
   git log --grep="interactions.jsonl"
   ```

3. **Create beads issue** (in beads repo):
   ```bash
   bd create \
     --title "Automatic agent interaction logging for Claude Code" \
     --type feature \
     --priority 2 \
     --label "claude" \
     --label "agent" \
     --label "audit"
   ```

4. **Reference this research**:
   - Link to gristle research doc
   - Include findings
   - Propose solution

## Files Created This Session

**In gristle repo** (for reference only):
- `docs/research/BEADS_AGENT_INTEGRATION_RESEARCH.md` - Detailed findings
- `docs/research/BEADS_NEXT_STEPS.md` - This file

**Action**: Copy these to beads repo or reference in beads issue

## Recommendation Summary

**DO**:
- ✅ Research in beads repo (existing work? planned features?)
- ✅ Engage with beads maintainers (issue/discussion)
- ✅ Start with manual protocol (low cost, immediate value)
- ✅ Build automation only if manual protocol proves valuable

**DON'T**:
- ❌ Build in gristle repo (wrong context)
- ❌ Build without checking upstream first (may duplicate work)
- ❌ Build automation before validating need (might not be valuable)

**TIMELINE**:
- Today: Research beads repo
- This week: Engage with beads community
- This month: Enhanced manual protocol + measurement
- Next month: Automation decision based on measured value

## Exit Criteria for This Session

This beads investigation is complete for gristle context:
- ✅ Research documented
- ✅ Assessment complete
- ✅ Next steps clear
- ✅ Work venue identified (beads repo)

**Next**: Return to gristle work OR switch to beads repo for continuation.
