<!-- docket:skill-pack:start -->
# Docket Skill Pack (Codex)

<!-- docket.skill.pack.version: docket.skills/v1 -->
<!-- docket.contract.hash: 4215e96e76b073e7c5b58adccdafa2958d65153bd2e869b3255f7560e863f2e0 -->
<!-- docket.skill.metadata.checksum: 4bbadff18330725650ed9e6233332d2f19ad7494eecfb23b9f4cb939b3b375fc -->
<!-- docket.skill.ids: ticket-discovery,ticket-authoring-apply,context-optimize,learning-replay,wrap-up-readiness -->

Use `docket start` to pick up prioritized ticket work.

### Skills
- `ticket-discovery` (required)
  - title: Discover Next Ticket
  - intent: planning
  - command: docket list --state open --format context
  - triggers: session_start, resume, task_selection
  - summary: Find the next actionable ticket and inspect its working context before coding.
- `ticket-authoring-apply` (required)
  - title: Transactional Ticket Authoring
  - intent: authoring
  - command: docket ticket scaffold --format json
  - triggers: multi_line_ticket_edit, bulk_ticket_changes, automation_mode
  - summary: Use scaffold/apply commands to author or update ticket specs without fragile shell quoting.
- `context-optimize` (optional)
  - title: Compact Ticket Brief
  - intent: context
  - command: docket context-optimize {ticket_id}
  - triggers: llm_context_budget, ticket_handoff, task_brief
  - summary: Generate a bounded brief from ticket context, learnings, and recent activity.
- `learning-replay` (optional)
  - title: Replay Relevant Learnings
  - intent: quality
  - command: docket learn replay {ticket_id}
  - triggers: pre_implementation, incident_recurrence, ticket_resume
  - summary: Replay top ranked learned rules for a ticket using the same ranking model as start.
- `wrap-up-readiness` (optional)
  - title: End-of-Session Wrap-Up
  - intent: review
  - command: docket wrap-up {ticket_id}
  - triggers: session_end, pre_review, handoff
  - summary: Run wrap-up readiness checks for AC completion, handoff quality, blockers, and review transition readiness.
<!-- docket:skill-pack:end -->
