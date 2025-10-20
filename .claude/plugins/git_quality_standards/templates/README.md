# Git Quality Standards - Templates

This directory contains ready-to-use templates for implementing git quality standards in any project.

## Quick Start

**5-Minute Setup** (from your project root):

```bash
# 1. Copy validation exceptions template
cp path/to/templates/.validation_exceptions.template .validation_exceptions

# 2. Copy markdownlint config
cp path/to/templates/.markdownlint.json.template .markdownlint.json

# 3. Copy and install git hooks
cp path/to/scripts/install-git-hooks.sh scripts/
cp path/to/scripts/pre-push-hook.sh scripts/
bash scripts/install-git-hooks.sh

# 4. Done! Git hooks are active.
```

---

## Templates Overview

### Configuration Files

#### `.validation_exceptions.template`

**Purpose**: Define files/patterns to exclude from validation

**Use when**: You have files that intentionally fail validation (tests, templates, archived docs)

**Customization**:
1. Copy to project root as `.validation_exceptions`
2. Uncomment sections relevant to your project
3. Add project-specific patterns
4. Keep minimal - prefer fixing issues over adding exceptions

**Example patterns**:
```bash
# Test files with intentional errors
tests/fixtures/*.md

# Archived documentation
deprecated/**
archived/**/*.md

# Template files with placeholders
templates/**

# Third-party files
vendor/**
node_modules/**
```

**Best practices**:
- Document WHY each exception exists (comments)
- Use specific paths over broad globs
- Review quarterly and remove unnecessary exceptions
- Prefer fixing issues to adding exceptions

---

#### `.markdownlint.json.template`

**Purpose**: Markdown linting configuration

**Use when**: You want consistent markdown formatting

**Customization**:
1. Copy to project root as `.markdownlint.json`
2. Adjust rules for your style guide
3. No changes needed for most projects (sensible defaults)

**Key rules in template**:
- MD013: Line length 100 chars (code blocks 120)
- MD033: Allow specific HTML elements (details, summary, etc.)
- MD041: Don't require H1 as first line (disabled)
- MD024: Allow duplicate headings in different sections

**Customization examples**:

```json
{
  "MD013": {
    "line_length": 120,  // Change to 120 if you prefer
    "code_blocks": false  // Disable line length in code blocks
  },
  "MD033": {
    "allowed_elements": [
      "details", "summary", "br",
      "table", "thead", "tbody"  // Add table elements
    ]
  }
}
```

**Install markdownlint**:
```bash
npm install -g markdownlint-cli
```

---

### Documentation Templates

#### `PR_DOCUMENTATION_TEMPLATE.md`

**Purpose**: Comprehensive PR documentation template

**Use when**: Creating pull requests

**Customization**:
1. Copy to `docs/prs/{branch-name}.md` for each PR
2. Fill in all sections (delete comment blocks)
3. Customize sections for your team's needs

**Structure**:
- **Summary**: What, Why, Impact
- **Epic Reference**: Link to epic/milestone
- **Type of Change**: Feature, bug fix, breaking, etc.
- **Files Changed**: Major changes and why
- **Quality Assurance**: Testing performed, coverage
- **Breaking Changes**: Migration guide if applicable
- **Deployment Notes**: Pre/post deployment steps
- **Security Considerations**: Security impact and mitigations
- **Dependencies**: New/updated dependencies
- **Rollback Plan**: How to revert if needed
- **Documentation Updates**: What docs were updated
- **Reviewer Notes**: Focus areas for review

**Customization tips**:

1. **Remove sections** you don't need:
   ```markdown
   <!-- For simple projects, remove:
   - Security Considerations
   - Deployment Notes
   - Rollback Plan
   -->
   ```

2. **Add custom sections**:
   ```markdown
   ## Performance Impact
   - Benchmark results
   - Resource usage changes

   ## Accessibility
   - WCAG compliance
   - Screen reader testing
   ```

3. **Simplify for small teams**:
   ```markdown
   ## Summary
   What changed and why

   ## Testing
   - Manual test 1
   - Manual test 2

   ## Checklist
   - [ ] Tested locally
   - [ ] Docs updated
   ```

**Location options**:
- `docs/prs/` (recommended)
- `.github/prs/`
- `docs/pull-requests/`

Update `pre-push-hook.sh` if using different location (line 96).

---

### CI/CD Templates

#### `github-actions-pr-checks.yml.template`

**Purpose**: Automated PR title and branch name validation

**Use when**: Using GitHub Actions

**Customization**:
1. Copy to `.github/workflows/pr-checks.yml`
2. Update `PROJECT_CODE` on line 30 (e.g., "CLD", "API", "WEB")
3. Update `EPIC_DIGITS` on line 31 if not using 4-digit epics
4. Customize `ALLOWED_TYPES` on line 37 if you have custom branch types

**What it validates**:
1. Branch name follows: `type/epic-XXX-9999-milestone-behavior`
2. PR title follows: `{prefix}(epic-XXX-9999): description`
3. PR title length under 72 characters (warning)
4. PR documentation exists (warning)

**Customization examples**:

```yaml
# 1. Add custom branch type
env:
  ALLOWED_TYPES: 'feature|fix|docs|style|refactor|test|chore|ci|hotfix'

# Add mapping in "Validate PR title" step:
case "$BRANCH_TYPE" in
  # ... existing cases ...
  hotfix)   PR_PREFIX="hotfix" ;;
esac

# 2. Change project code pattern (2-letter codes)
# Line 72: Change {2,4} to {2,2}
BRANCH_REGEX="^(${{ env.ALLOWED_TYPES }})/epic-[A-Z]{2,2}-[0-9]{4}-.+"

# 3. Make PR docs check mandatory
# Line 169: Change exit 0 to exit 1
exit 1  # Fail if PR docs missing

# 4. Validate on more branches
on:
  pull_request:
    branches: [ main, master, develop, staging ]
```

**Testing**:
1. Push workflow file to `.github/workflows/`
2. Create test PR with correct format → should pass
3. Create test PR with wrong format → should fail with helpful error
4. Review error messages for clarity

---

#### `github-actions-validation.yml.template`

**Purpose**: Repository structure and quality validation

**Use when**: Using GitHub Actions for CI/CD

**Customization**:
1. Copy to `.github/workflows/validation.yml`
2. Update required files list (lines 96-99)
3. Update required directories list (lines 145-148)
4. Customize trigger branches (lines 11-14)

**What it validates**:
1. Markdown syntax (using markdownlint)
2. Required files exist (README.md, .gitignore, etc.)
3. Required directories exist (.github, etc.)
4. Git hooks configuration
5. Validation exceptions file

**Customization examples**:

```yaml
# 1. Add required files for Node.js project
required_files=(
  "README.md"
  ".gitignore"
  "package.json"
  "package-lock.json"
)

# 2. Add required directories
required_dirs=(
  ".github"
  "src"
  "tests"
  "docs"
)

# 3. Add custom validation job
validate-code:
  name: 'Validate Code'
  runs-on: ubuntu-latest
  steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-node@v4
    with:
      node-version: '18'
  - run: npm install
  - run: npm run lint
  - run: npm run type-check

# 4. Update summary job dependencies
needs: [
  validate-markdown,
  validate-structure,
  validate-git-setup,
  validate-code  # Add new job
]

# 5. Add result to summary
if [[ "${{ needs.validate-code.result }}" == "success" ]]; then
  echo "✅ **Code Validation**: Passed" >> $GITHUB_STEP_SUMMARY
else
  echo "❌ **Code Validation**: Failed" >> $GITHUB_STEP_SUMMARY
fi
```

**Advanced customization**:

1. **Add Python validation**:
   ```yaml
   - uses: actions/setup-python@v5
     with:
       python-version: '3.11'
   - run: pip install pylint black
   - run: black --check .
   - run: pylint src/
   ```

2. **Add security scanning**:
   ```yaml
   - uses: aquasecurity/trivy-action@master
     with:
       scan-type: 'fs'
       scan-ref: '.'
   ```

3. **Add test coverage**:
   ```yaml
   - run: npm test -- --coverage
   - run: |
       COVERAGE=$(cat coverage/coverage-summary.json | jq '.total.lines.pct')
       if (( $(echo "$COVERAGE < 80" | bc -l) )); then
         echo "❌ Coverage $COVERAGE% is below 80%"
         exit 1
       fi
   ```

---

## Adoption Workflow

### For New Projects

```bash
# 1. Create project structure
mkdir -p .github/workflows docs/prs scripts

# 2. Copy all templates
cp path/to/templates/.validation_exceptions.template .validation_exceptions
cp path/to/templates/.markdownlint.json.template .markdownlint.json
cp path/to/templates/PR_DOCUMENTATION_TEMPLATE.md docs/

# 3. Copy GitHub Actions workflows
cp path/to/templates/github-actions-pr-checks.yml.template .github/workflows/pr-checks.yml
cp path/to/templates/github-actions-validation.yml.template .github/workflows/validation.yml

# 4. Copy git hooks
cp path/to/scripts/*.sh scripts/
bash scripts/install-git-hooks.sh

# 5. Customize templates
# - Edit .github/workflows/pr-checks.yml: set PROJECT_CODE
# - Edit .github/workflows/validation.yml: set required files/dirs
# - Edit .validation_exceptions: uncomment needed patterns

# 6. Install dependencies
npm install -g markdownlint-cli

# 7. Test
bash scripts/validate-repository.sh

# 8. Commit
git add .
git commit -m "chore: add git quality standards"
git push
```

### For Existing Projects

```bash
# 1. Backup existing configs
cp .gitignore .gitignore.backup
cp .github/workflows/ci.yml .github/workflows/ci.yml.backup

# 2. Copy templates (don't overwrite existing)
cp -n path/to/templates/.validation_exceptions.template .validation_exceptions
cp -n path/to/templates/.markdownlint.json.template .markdownlint.json

# 3. Merge GitHub Actions
# - Manually merge with existing workflows
# - Or rename: pr-checks.yml → git-quality-pr-checks.yml

# 4. Install hooks (check existing .git/hooks/ first)
ls -la .git/hooks/
cp path/to/scripts/pre-push-hook.sh scripts/
bash scripts/install-git-hooks.sh

# 5. Customize for existing conventions
# - Update PROJECT_CODE in workflows
# - Add existing files to .validation_exceptions if needed
# - Adjust branch patterns to match existing branches

# 6. Test without failing
# - Run validation: bash scripts/validate-repository.sh
# - Fix or add exceptions for failures
# - Iterate until passing

# 7. Gradual rollout
# - Start with warnings only (change exit 1 → exit 0)
# - Give team time to adapt
# - Make failures after 2-week grace period
```

---

## Template Maintenance

### Versioning

All templates include version in header:
```markdown
# Template Version: 1.0.0
# Skill: git_quality_standards
# Last Updated: 2025-01-20
```

**When to update version**:
- Breaking changes: bump major (1.0.0 → 2.0.0)
- New features: bump minor (1.0.0 → 1.1.0)
- Bug fixes: bump patch (1.0.0 → 1.0.1)

### Keeping Templates Updated

```bash
# 1. Periodic review (quarterly)
# - Check for new best practices
# - Review validation exceptions
# - Update tool versions

# 2. When git_quality_standards skill updates
# - Re-copy templates
# - Merge with your customizations
# - Test thoroughly

# 3. Version tracking
# - Note template version in commit message
# - Track customizations separately
# - Document deviations from template
```

---

## Common Issues

### Markdownlint Fails on Valid Markdown

**Problem**: Markdown looks fine but linting fails

**Solutions**:
1. Check specific rule number in error (e.g., MD013)
2. Adjust rule in `.markdownlint.json`
3. Or add file to `.validation_exceptions`

```bash
# See specific rule details
markdownlint --help | grep MD013

# Test against single file
markdownlint path/to/file.md

# Show which rule failed
markdownlint -o markdownlint-results.json path/to/file.md
cat markdownlint-results.json
```

### GitHub Actions Not Triggering

**Problem**: Pushed changes but workflow didn't run

**Solutions**:
1. Check workflow file syntax: https://www.yamllint.com/
2. Verify branch pattern matches: `branches: [ main ]`
3. Check file location: must be in `.github/workflows/`
4. Verify workflow is enabled in GitHub Settings → Actions

```bash
# Validate YAML syntax locally
yamllint .github/workflows/pr-checks.yml

# Check file is in correct location
ls -la .github/workflows/

# View workflow status
gh workflow list
gh workflow view pr-checks.yml
```

### PR Documentation Check Always Fails

**Problem**: PR docs exist but hook/workflow can't find them

**Solutions**:
1. Verify filename matches branch name with slashes → dashes
2. Check directory: `docs/prs/` vs `.github/prs/`
3. Ensure file committed to branch (not just local)

```bash
# Check expected filename
BRANCH=$(git branch --show-current)
EXPECTED=$(echo "$BRANCH" | sed 's/\//-/g')
echo "Expected: docs/prs/${EXPECTED}.md"

# Verify file exists and is committed
git ls-files docs/prs/

# Check file in remote
git ls-tree HEAD docs/prs/
```

### Validation Exceptions Not Working

**Problem**: File in `.validation_exceptions` still being validated

**Solutions**:
1. Check path is relative to repo root (not absolute)
2. Verify no typos in path
3. Ensure no trailing whitespace
4. Test glob pattern

```bash
# Test if pattern matches files
PATTERN="deprecated/*"
find . -path "./$PATTERN" -name "*.md"

# Check for trailing whitespace
cat -A .validation_exceptions | grep -E ' $'

# Verify file format
file .validation_exceptions  # Should be "ASCII text"
```

---

## Integration Examples

### GitLab CI

```yaml
# .gitlab-ci.yml
validate:
  stage: test
  image: node:18
  before_script:
    - npm install -g markdownlint-cli
  script:
    - bash scripts/validate-repository.sh
```

### Azure DevOps

```yaml
# azure-pipelines.yml
- task: NodeTool@0
  inputs:
    versionSpec: '18.x'
- script: npm install -g markdownlint-cli
- script: bash scripts/validate-repository.sh
  displayName: 'Validate Repository'
```

### Jenkins

```groovy
// Jenkinsfile
stage('Validate') {
  steps {
    sh 'npm install -g markdownlint-cli'
    sh 'bash scripts/validate-repository.sh'
  }
}
```

### Pre-commit Framework

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: git-quality-validation
        name: Git Quality Validation
        entry: bash scripts/validate-repository.sh
        language: system
        pass_filenames: false
```

---

## Support

For issues, questions, or contributions:

1. Check main SKILL.md documentation
2. Review troubleshooting sections
3. Test with minimal reproduction
4. Check template version matches skill version

**Template version**: 1.0.0
**Skill version**: See `../SKILL.md` frontmatter
**Last updated**: 2025-01-20
