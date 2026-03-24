<!-- docket:start -->
## Docket Workflow

- Use `docket list --state open --format context` to pick work.
- Use `docket show TKT-NNN --format context` before coding.
- Use `docket update TKT-NNN --state in-progress` when moving a ticket into active work.
- Use `docket ac add` / `docket ac complete` for acceptance tracking.
- Add `Ticket: TKT-NNN` trailer to commit messages.
<!-- docket:end -->
