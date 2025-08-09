# Spec-Driven Development (SDD) — Kiro-style Guideline

## Goals
- **Clarity**: Understand problems before coding.
- **Alignment**: Stakeholders review specs first.
- **Separation**: Strict split of requirements / design / tasks.
- **Living Docs**: `.claude/docs/spec/` is the source of truth.

## spec Structure
Use repo root `.claude/docs/spec/{requirements,design,tasks}.md`.

### requirements.md — **What**
- **Use EARS** for all functional requirements.
- Include: `REQ-X.Y` IDs, stakeholders & goals, constraints, verifiable behaviors.
- Never include: tasks, technical details, architecture, estimates.

**EARS patterns**
- Always: `The system shall <response>`
- Event: `When <trigger>, the system shall <response>`
- State: `While <state>, the system shall <response>`
- Conditional: `Where <condition>, the system shall <response>`

### design.md — **How**
- Architecture overview, interfaces & data contracts, resilience, security.
- ADRs with rationale & alternatives.
- Never include: “shall/should/must” requirements or task checklists.

### tasks.md — **Execution**
- Markdown checkboxes only: `- [ ]`
- IDs: `TASK-###` (sequential, non-reused; retired tasks move to `Retired`).
- **refs:** one or more `REQ-…` per task (mandatory).
- `DoD:` optional (mandatory only if tool option `--require-dod` is used).
- Subtasks as indented `- [ ]`.

**Example**
```
* [ ] TASK-010 Create rating display · refs: REQ-1.1, REQ-2.1 · DoD: unit tests cover stars
  * [ ] Implement component
  * [ ] Add keyboard/mouse interaction
```

## Workflow
1) Discover/update `.claude/docs/spec/`; create if missing.
2) Draft/update `requirements.md` (EARS only).
3) Draft/update `design.md` (architecture & rationale).
4) Draft/update `tasks.md` (checkbox + refs + optional DoD).
5) Execute tasks; move to `Done` when complete; keep docs in sync.
6) Use **refine** to clarify/shorten without scope change.

## ID Policies
- **REQ**: `REQ-X.Y` (X=category, Y=sequence). Authors manage; avoid gaps/dupes.
- **TASK**: `TASK-###` auto-issued by tool; never reuse retired numbers.

## Anti-Patterns
- Requirements in design; design analysis in tasks; task-like text in requirements.
- Copying same content across files; time estimates inside tasks.md.
