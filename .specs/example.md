---
id: example-task
title: "Example Task: Demonstrate Spec Format"
priority: 5
depends_on:
  - setup-task
  - init-task
test_cmd: "go test ./..."
requires_approval: true
---
This is an example task specification demonstrating all available frontmatter fields.

## Description

Task specs use YAML frontmatter (between `---` markers) for metadata, followed by
a markdown body for the detailed specification.

## Frontmatter Fields

| Field              | Type       | Required | Description                           |
|--------------------|------------|----------|---------------------------------------|
| `id`               | string     | yes      | Unique task identifier                |
| `title`            | string     | yes      | Human-readable task title             |
| `priority`         | int        | no       | Higher = more important (default: 0)  |
| `depends_on`       | []string   | no       | IDs of tasks that must complete first |
| `test_cmd`         | string     | no       | Command to verify task completion     |
| `requires_approval`| bool       | no       | Whether human approval is needed      |

## Usage

```bash
# Validate spec files
volta spec validate --dir .specs/

# Sync to database (dry run)
volta spec sync --dir .specs/ --project myproject --dry-run

# Sync and release draft tasks
volta spec sync --dir .specs/ --project myproject --release
```

## Acceptance Criteria

- [ ] Spec file parses without errors
- [ ] All frontmatter fields are recognized
- [ ] Dependencies are resolved correctly
