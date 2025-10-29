# chore(epic-TSE-0001): standardize git quality standards infrastructure

## Summary

This PR standardizes the git quality standards infrastructure in audit-data-adapter-go by updating validation scripts to use simplified PR documentation matching and TODO file handling.

**Key Changes:**
- Updated `.claude/plugins/git_quality_standards/` scripts to latest versions
- Simplified PR documentation matching to use exact branch name with slash-to-dash conversion
- Updated validation scripts to accept both TODO.md and TODO-MASTER.md
- Synchronized with ecosystem-wide git standards

## What Changed

### Git Quality Standards Updates

**Updated Scripts:**
- `pre-push-hook.sh` - Simplified PR matching logic, removed PR prefix mapping
- `validate-all.sh` - Added TODO.md OR TODO-MASTER.md check, restored PR documentation validation
- `validate-repository.sh` - Updated to match ecosystem standards
- `install-git-hooks-enhanced.sh` - Latest version with improved installation logic

**PR Documentation Matching (Simplified):**
```bash
# Old (removed): Complex PR prefix mapping and epic pattern matching
# New (simple): Exact branch name with slashes converted to dashes
BRANCH_FILENAME=$(echo "$CURRENT_BRANCH" | sed 's/\//-/g')
Expected file: docs/prs/${BRANCH_FILENAME}.md
```

**TODO File Handling:**
- Accepts either TODO.md or TODO-MASTER.md
- Supports TODO journal system with TODO-HISTORY.md archives

### Repository Context

This is part of epic TSE-0001 Foundation work to establish consistent git workflows, validation standards, and documentation practices across all repositories in the trading ecosystem.

## Test Plan

- [x] Run `bash scripts/validate-all.sh` - passes all checks
- [x] Verify PR documentation exists for current branch
- [x] Confirm markdown linting passes
- [x] Verify TODO.md exists and is valid
- [x] Test pre-push hook logic (dry run)

## Related Issues

- Epic TSE-0001: Foundation - Git Quality Standards
- Standardizing validation scripts across 9+ repositories

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
