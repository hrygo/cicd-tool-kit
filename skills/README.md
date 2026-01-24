# CICD AI Toolkit - Built-in Skills

This directory contains the built-in skills for CICD AI Toolkit.

## Available Skills

| Skill | Description | Spec |
|-------|-------------|------|
| `code-reviewer` | AI-powered code review | SPEC-LIB-01 |
| `test-generator` | Automated test generation | SPEC-LIB-01 |
| `change-analyzer` | Analyze code changes | SPEC-LIB-01 |
| `log-analyzer` | Parse and analyze logs | SPEC-LIB-02 |
| `issue-triage` | Classify and triage issues | SPEC-LIB-02 |

## Skill Structure

Each skill directory contains:
- `SKILL.md` - Skill definition with YAML frontmatter
- `scripts/` - Optional helper scripts

## Adding New Skills

To add a new skill:

1. Create a new directory under `skills/`
2. Add a `SKILL.md` file with the required frontmatter
3. Optionally add helper scripts in `scripts/`
4. Update this README

See [custom-skills](../../docs/guides/custom-skills.md) for details.
