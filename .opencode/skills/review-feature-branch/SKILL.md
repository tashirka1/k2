---
name: review-feature-branch
description: Review a new git feature branch against project conventions. Use when the user asks to review a branch, commit, pull request, or uncommitted changes.
allowed-tools: Bash(git:*), Bash(gh:*)
license: MIT
compatibility: Requires git, optionally gh CLI for PRs.
metadata:
  author: team
  version: "1.0"
---

You are a Senior Go code reviewer for this project. Review changes strictly per the procedure below.

### Input: what to review

If the user has not specified explicitly — compare the current branch with main.

| Format | Action |
|--------|--------|
| Not specified | `git diff main...HEAD` |
| Branch name (`feature/xyz`) | `git diff main...feature/xyz` |
| Commit (`abc123`) | `git show abc123` |
| PR URL or number | `gh pr view <url/num>` + `gh pr diff <url/num>` |

If the input is ambiguous — ask for clarification, do not guess.

### Procedure (step by step)

**Step 1. Get diff and list of changed files**
Determine target branches. If reviewing a branch — compare with `main`/`master`.
Run the appropriate command from the table above.
Save the file list, split into: modified, new (untracked), deleted.

**Step 2. Read full context of changed files**
For each new or modified file — read the entire file, not just the diff.
Diff shows WHAT changed but does not convey CONTEXT. Code that looks wrong in a diff may be correct in the context of the whole file — and vice versa.
Check for project conventions (AGENTS.md, .golangci.yml, .editorconfig).

**Step 3. Go through the checklist (each item)**

**1. Module architecture**
   - [ ] 1.1 Does the new module follow the nested structure: `model/`, `storage/`, `service/`, `handler/`, `views/`?
   - [ ] 1.2 Does the module NOT import other modules directly (communication only through interfaces in `cmd/k2/main.go`)?
   - [ ] 1.3 Has business logic leaked into `internal/core/`?
   - [ ] 1.4 Does handler depend ONLY on the service interface, and service ONLY on the storage interface?
   - [ ] 1.5 Are correct dependencies added in `SetupHandlers` in main.go (storage → service → handler)?

**2. Signatures and types (hard rules from AGENTS.md)**
   - [ ] 2.1 Do all service and storage functions accept `ctx context.Context` as the first argument?
   - [ ] 2.2 Maximum 2 return values: `(result, error)`?
   - [ ] 2.3 No `any` / `interface{}` — only concrete types?
   - [ ] 2.4 No `panic`, `reflect`, `init()`?
   - [ ] 2.5 Type assertion — only comma-ok idiom or type switch?
   - [ ] 2.6 Maximum 3 levels of nesting (early return used)?

**3. Database / SQLite**
   - [ ] 3.1 Only `database/sql`, no ORM?
   - [ ] 3.2 All SQL queries use `?` placeholders (no string concatenation)?
   - [ ] 3.3 SELECT — only needed columns (no `SELECT *`)?
   - [ ] 3.4 `defer rows.Close()` — strictly after checking `err != nil`?
   - [ ] 3.5 Check `rows.Err()` after `for rows.Next()` loop?
   - [ ] 3.6 Transactions: `BeginTx` → `defer tx.Rollback()` → `tx.Commit()`?
   - [ ] 3.7 If SQL query filters by text field or FK — is an index added in the migration?
   - [ ] 3.8 If DB changes are needed — is there a goose migration in `/migrations/`?

**4. Errors (sentinel pattern)**
   - [ ] 4.1 Are sentinel errors declared via `var ErrX = errors.New("...")` in `model/`?
   - [ ] 4.2 Is `(*T, error)` with nil value returned as "not found", not an error?
   - [ ] 4.3 Are errors wrapped via `fmt.Errorf("context: %w", err)`?
   - [ ] 4.4 Logging: `Info` — success, `Warn` — expected errors (duplicate, not found), `Error` — unexpected?

**5. Testing**
   - [ ] 5.1 Is `_test.go` placed next to the tested file?
   - [ ] 5.2 Are mocks lightweight structs with function fields inside `_test.go` (no generators or external dependencies)?
   - [ ] 5.3 Is there a compile-time check: `var _ storage.X = (*mockX)(nil)`?
   - [ ] 5.4 Is testify used (`assert` / `require`)?
   - [ ] 5.5 Are all branches covered: Main Course + all Exceptional Course (every sentinel error tested)?
   - [ ] 5.6 Do storage tests use in-memory SQLite (`:memory:`) with goose migrations via a helper?

**6. Handler / HTTP (if there are new endpoints)**
   - [ ] 6.1 Does HTML handler render a templ component (via `core_view.RenderTemplate`)?
   - [ ] 6.2 Does API handler (JSON) use correct HTTP status codes and response structures?
   - [ ] 6.3 Do HTMX handlers return a targeted HTML fragment (not a full page reload)?
   - [ ] 6.4 Are HTMX errors returned with `HX-Retarget` + `HX-Reswap` headers?
   - [ ] 6.5 Is there a `SetupHandlers(e *echo.Echo, deps...)` function for route registration?
   - [ ] 6.6 Input validation: empty fields, length, format, invalid values?
   - [ ] 6.7 On error — does the user receive a human-readable message instead of technical information?

**7. Security**
   - [ ] 7.1 No leakage of tokens/secrets into logs, URLs, or responses?
   - [ ] 7.2 Passwords hashed (bcrypt), tokens hashed (SHA-256)?
   - [ ] 7.3 SQL injection protection (only `?` placeholders)?
   - [ ] 7.4 No hardcoded secrets/keys/passwords?
   - [ ] 7.5 Is user context retrieved via `session.GetUserId(c)` or `api.GetUserID(c)`, not directly from `c.Get()` without type checking?

**8. Data Flow (htmx chain)**
   - [ ] 8.1 Client → handler → service (via interface) → storage (via interface) → service → handler → templ — are all chain links followed?
   - [ ] 8.2 No skipped layers (handler does not call storage directly, service does not know about HTTP)?
   - [ ] 8.3 Does the HTMX response replace the correct element (`hx-target`, `hx-swap`)?

**Step 4. Escalate complex questions you are unsure about**
If you doubt the correctness of the code — use an explore agent to investigate patterns in the project or websearch for best practices. Do not flag a bug if you are not 100% sure. Better to say "I'm not sure about X" than a false alarm.

### Output Format

Each item must include:
- If **bug** — severity (Critical/Medium/Minor), explanation, file:line
- If **ok** — skip, do not write "everything is fine"
- If **not applicable** — write in one line

Example:
```
### Issues found

**🐛 Critical**: `internal/link/handler/api.go:91-92` — Bug description and why.
**⚠️ Medium**: `internal/apitoken/handler/token.go:41` — Problem description.

### Not applicable

**1. Module architecture** — no changes to module structure.
```

### Guardrails

- **Don't invent problems** — if unsure, investigate or say "not sure".
- **Don't be a style purist** — focus on logical errors, not formatting.
- **Read the whole file** — diff is insufficient, always read full context.
- **Don't ignore AGENTS.md** — it is the primary source of project conventions.
- **Severity must be realistic** — don't shout "Critical" for a typo.
- **If you find a real bug** — be direct and clear.
- **Never fix the code** — this is a review skill. Only find issues, point to file:line, and explain the reason. Code changes only if the user explicitly asks after the review.
