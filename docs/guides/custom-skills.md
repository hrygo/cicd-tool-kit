# Custom Skills

## Skill Definition

Skills are defined using `SKILL.md` files with YAML frontmatter:

```markdown
---
name: "my-skill"
version: "1.0.0"
description: "My custom skill"
author: "your-name"
options:
  thinking:
    budget_tokens: 4096
  tools:
    allow:
      - "read"
      - "grep"
---

# My Skill

Description of what this skill does...
```

## Frontmatter Reference

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique skill identifier |
| `version` | string | Semantic version |
| `description` | string | Skill description |
| `author` | string | Author name |
| `options.thinking.budget_tokens` | int | Max tokens for thinking |
| `options.tools.allow` | list | Allowed tools |
| `options.tools.deny` | list | Denied tools |

## Output Format

Skills should output structured XML:

```xml
<result>
  <summary>Brief summary</summary>
  <details>Detailed findings</details>
</result>
```

## Skill Locations

Skills are loaded from:
1. `./skills/` (project-local)
2. `~/.cicd-ai-toolkit/skills/` (user-global)
3. Built-in skills

## Helper Scripts

Add helper scripts in `scripts/` subdirectory:

```bash
skills/
  my-skill/
    SKILL.md
    scripts/
      helper.sh
```

Use in skill prompt:

```markdown
Run: ./scripts/helper.sh files
```
