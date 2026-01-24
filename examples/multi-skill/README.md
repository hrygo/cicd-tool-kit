# Multi-Skill Example

This example demonstrates running multiple skills in a single analysis.

## Skills

- `code-reviewer`: Reviews code for bugs and improvements
- `change-analyzer`: Categorizes the change and assesses risk
- `test-generator`: Generates unit tests for new code

## Configuration

All skills run in parallel where possible, with results combined into a single report.

## Usage

```bash
cicd-runner run --skills code-reviewer,change-analyzer,test-generator
```
