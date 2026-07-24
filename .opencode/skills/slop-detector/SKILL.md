---
name: slop-detector
description: Checks a Go project for signs of "AI slop" — stylistic patterns characteristic of AI-generated code.
allowed-tools: Bash(git:*), Bash(gh:*), Bash, Read, Grep, Glob, WebSearch, WebFetch
license: MIT
compatibility: Requires git, optionally gh CLI for PRs.
metadata:
  author: team
  version: "1.0"
---

You are a Senior Go code reviewer for this project. Review changes strictly per the procedure below.

### Input

By default — all `.go`, `.css`, `.js` and `.templ` files in the project.
Optional: specify a path with `--path=internal/link/` to check a specific directory.

### Procedure (step by step)

**Step 0. Determine scope**
If `--path` is specified — work only within it. If not — check the project root (recursively, excluding `vendor/`).

**Step 1. Collect metrics (bash)**
For each checklist item, collect data via bash/grep/rg. Save results for analysis.

**Step 2. Analyze checklist (each item)**
Go through each item. If a threshold is exceeded — record the issue with severity and file:line.

**Step 3. Output report**

### Checklist

#### 1. Comment Noise

**1.1 Total comment volume**
- `find . -name '*.go' | xargs wc -l` — total LOC
- `rg '^\s*//' --type go | wc -l` — comment lines
- If comment percentage > 30% of total LOC — Medium

**1.2 Obvious comments**
- Search patterns: `// increment`, `// decrement`, `// return`, `// check if`, `// set the`, `// get the`, `// do nothing`, `// this function`
- `rg '(// (increment|decrement|return|check if|set the|get the|this function))' --type go` 
- Each match — Low, file:line

**1.3 Doc comments on unexported functions**
- Idiomatic Go: doc comments (`// Foo`) — only for exported functions
- Find `^(// [a-z]|[ \t]+// [a-z])` near `func [a-z]` — AI-style indicator
- `rg '(// [a-z].*\nfunc [a-z])' --type go --multiline` 
- Each match — Low

**1.4 Excessive inline comments on trivial operations**
- Search `//.*(err|nil|true|false|ok|error|result)` on lines with trivial assignments
- `rg '(err|result|ok).*(//.*(err|result|ok|nil|return))' --type go`
- Medium if >5 matches per 1000 LOC

#### 2. Naming

**2.1 Abnormally long identifiers**
- Search names longer than 30 chars: `rg -P '\b[a-zA-Z_][a-zA-Z0-9_]{30,}\b' --type go`
- Each match — Medium, file:line
- Ignore imports (package path) and URLs in tests

**2.2 Abstract generic names**
- Search `data`, `item`, `result`, `value`, `input`, `output`, `info`, `helper`, `util`, `manager` as parameter/variable names
- `rg '(func.* (data|item|result|value|input|output|info|helper|util|manager) | (data|item|result|value|input|output|info) :=)' --type go`
- Low — each match, Medium — if >10 per file

**2.3 Hungarian notation / non-Go naming**
- Search snake_case in Go functions (not tests): `rg '[a-z]+_[a-z]+[a-z]+\b' --type go`
- Low — each match in function body (not imports, not tests)
- Ignore: test names (`TestXxx`), constants, test files

**2.4 Template AI names**
- Search patterns: `myVariable`, `exampleData`, `sampleText`, `dummyValue`, `testFunc`, `tempVar`
- `rg '(myVariable|exampleData|sampleText|dummyValue|testFunc|tempVar|placeholderValue)' --type go`
- Low — each match

#### 3. Types & Signatures

**3.1 Use of `any` / `interface{}`**
- `rg '\bany\b' --type go` and `rg '\binterface\{\}\b' --type go`
- Exceptions: `Error` field in JSON responses, signal.NotifyContext, Printf/printf-like
- Medium — if >5 non-trivial usages
- High — if `any` used in own function signatures (not calls to third-party APIs)

**3.2 Use of `reflect`**
- `rg '"reflect"' --type go`
- Critical — each usage (forbidden by AGENTS.md)

**3.3 Use of `panic` / `init()`**
- `rg '\bpanic\b' --type go`, `rg '^func init\b' --type go`
- Critical — each usage (forbidden by AGENTS.md)

**3.4 Functions with 3+ return values**
- `rg '^func.*\(.*,.*,.* .*\)' --type go` (rough check: 3+ values in return)
- Medium — each function with 3+ return values (except constructors with error)

**3.5 Functions with 6+ parameters**
- `rg '^func \([^)]+\) [^(]+\([^)]+,[^)]+,[^)]+,[^)]+,[^)]+,' --type go` 
- Medium — each such function

#### 4. Architecture Anomalies

**4.1 Monotonous rhythm (uniform function length)**
- Collect all function lengths in each file: bash + awk
- If a file has 5+ functions and the difference between min and max length ≤ 8 lines — Medium
- AI indicator: all functions end up roughly the same size

**4.2 Deep nesting (>4 levels)**
- `rg '^\s{20,}' --type go` (4 levels = 20 spaces with tab-stop 4)
- Medium — each match (violates AGENTS.md, max 3 levels)

**4.3 Identical error handling pattern in all functions**
- If a file has 8+ functions and all have a bare `if err != nil { return err }` without `fmt.Errorf` — Low
- AI indicator: does not wrap errors with context
- `rg '(if err != nil \{\s*return err\s*\})' --type go --multiline`

**4.4 Unnecessary single-implementation interfaces**
- Find interfaces with exactly 1 implementation (struct) in the same package
- grep: find all `type Foo interface` and check number of implementations
- Medium — interface not imported from other packages with 1 implementation
- Low — interface imported from 1 package with 1 implementation

#### 5. Hallucinations & Deprecated APIs

**5.1 Deprecated packages**
- Search imports: `"io/ioutil"`, `"golang.org/x/net/context"`, `"errors.Wrap"`, `"github.com/pkg/errors"`
- High — each usage of a deprecated package

**5.2 Nonexistent common methods**
- Search: `.ToLower()`, `.ToUpper()`, `.hasOwnProperty()` (JS in Go), `.toString()` (Java in Go)
- In Go: `strings.ToLower`, `strings.ToUpper`
- High — each match

**5.3 Incorrect sqlite constructor**
- Search: `sql.Open("sqlite"` or `sql.Open("sqlite3", ...)` without `_ "github.com/mattn/go-sqlite3"`
- High — if driver imported but wrong open string, or open string without driver

#### 6. Defensive Overkill

**6.1 `len(x) > 0` before `for range`**
- Search pattern: `if len\(.*\) > 0\s*\{` on line preceding `for range`
- Low — each match (range over slice/map is safe, check is redundant)

**6.2 Redundant nil check before range**
- Search: `if .* != nil\s*\{` preceding `for range .*\{`
- Low — each match (range over nil map/slice is safe)

**6.3 Double error check**
- Two `if err != nil` in a row in the same function without changing `err` between them
- Low — each match

### Output Format

```
### slop-detector report

**🐛 Critical**: `internal/foo/bar.go:42` — Use of `reflect` (violates AGENTS.md)
**🐛 High**: `internal/foo/service.go` — Use of deprecated package `"io/ioutil"`
**⚠️ Medium**: `internal/foo/handler.go` — Comment noise: 42% of lines are comments (threshold 30%)
**⚠️ Medium**: `internal/foo/api.go:15` — Identifier `processedDataListResult` (35 chars) looks AI-generated
**ℹ️ Low**: `internal/foo/util.go:33` — Doc comment on unexported function `parseInput`
**ℹ️ Low**: `internal/foo/handler.go:88` — Redundant check `len(x) > 0` before `for range`

### Not applicable / no issues

Categories where thresholds are not exceeded, or the check yielded no results.
```

### Guardrails

- **Don't invent problems** — if the result is ambiguous, skip it.
- **Consider project context** — `any` in an Echo JSON handler is fine. `any` in business logic is suspicious.
- **Severity must be realistic** — `Critical` only for `panic`, `reflect`, `init()`.
- **Don't check `vendor/`** — only project code.
- **Avoid false positives** — if in doubt, better to skip than to give a false positive.
- **Never fix the code** — this is a review skill. Only find issues, point to file:line, and explain the reason. Code changes only if the user explicitly asks after the review.

### Reference

**Thresholds:**
- Comment noise: >30% comment lines of total LOC = Medium
- Identifier length: >30 chars = Medium
- `any`/`interface{}`: >5 non-trivial usages = Medium
- Abstract names: >10 per file = Medium
- Functions with 3+ returns: each match = Medium
- Monotonous rhythm: min-max diff ≤ 8 lines with 5+ functions = Medium
- Obvious comments: each match = Low

**Severity:**
- `Critical` — violates AGENTS.md hard prohibitions (panic, reflect, init)
- `High` — serious architectural or security issue (deprecated API, hallucinations)
- `Medium` — probable AI pattern (comment noise, long names, nesting)
- `Low` — suspicious but could be coincidence (snake_case, redundant guard)
