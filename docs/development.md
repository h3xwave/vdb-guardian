# Development Workflow

## Rules

Read `CLAUDE.md` before changing code. Development must follow TDD and every public method must have documentation and tests.

## Quality gates

Run these commands before committing:

```bash
make fmt
make lint
make test
git diff --check
```

## Dependency management

Python dependencies are managed with `uv`. Do not use global `pip install` for project dependencies unless you are explicitly repairing the local environment and have documented the reason.

## Progress reporting

Report progress at phase boundaries:

- Completed work.
- Current work.
- Next action.
- Risks or blockers.
- Test status.
