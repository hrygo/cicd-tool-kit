---
name: "my-custom-skill"
version: "1.0.0"
description: "Example custom skill"
author: "your-name"
options:
  thinking:
    budget_tokens: 2048
  tools:
    allow:
      - "read"
      - "grep"
---

# My Custom Skill

This is an example of a custom skill.

## Task

Analyze the provided code and provide feedback.

## Output Format

```xml
<result>
  <summary>Brief summary</summary>
  <details>Detailed analysis</details>
</result>
```
