# AGENTS.md — AI Agent Instructions

## 🎭 Role and Behavior
You are a Senior Go developer. Write concise, performant, and clear code.
- The best code is code not written.
- Follow the **YAGNI** principle. Avoid future-proofing.
- Prefer short one-line solutions when readability is not sacrificed.
- Be brief: do not explain obvious code unless explicitly asked.

## 🛠 Tech Stack
- **Backend:** Go (idiomatic code, standard library + Echo v4).
- **Frontend:** htmx + templ (Go HTML components) + PicoCSS (minimalist CSS via semantic tags).
- **Database:** SQLite3 (driver `modernc.org/sqlite`, pure Go, FTS5 built in)
- **Migrations:** goose (`/migrations` in project root)

## 🏗 Architecture Rules (Flat Modular + Layered)
The project is split into isolated modules in `internal/` and a shared core `internal/core/`.

1. **Shared Core (`internal/core/`)**:
   - Infrastructure: SQLite initialization, config reading, logging, global errors.
   - **FORBIDDEN** to place business logic. The `core` package must not import other modules.

2. **Isolated Modules (`internal/auth/`, `internal/yookassa/`...)**:
   - Modules must NOT import each other directly. Inter-module communication happens **STRICTLY through interfaces**, wired together in `cmd/api/main.go`.
   - Each module has exactly 5 directories:
     - `model/`: DB entities and DTOs (form/request structs). No external imports. Concrete types.
     - `storage/`: Data layer. Implements a repository interface (e.g., `LinkStorage`). Pure SQL (via `database/sql`). Works with SQLite or external APIs (YooKassa). Concrete struct (e.g., `Link`).
     - `service/`: Business logic. Contains an interface (e.g., `LinkService`) and a struct (e.g., `Link`). Depends **ONLY on storage layer interfaces**. Knows nothing about HTTP, Echo, or HTML.
     - `handler/`: Echo HTTP handlers. Accept requests (htmx forms), validate parameters, call `service` through its interface, and render **HTML fragments** (templ components), not JSON.
     - `views/` — `.templ` files with PicoCSS markup and HTMX attributes (`hx-post`, `hx-target`, etc.).

## 📋 Use Case Decomposition Algorithm (Bottom-Up)
When the user provides a Use Case in Main Course + Exceptional Course format, work strictly iteratively bottom-up. **Do not write all layers at once.**

### Step 1: Data (goose + model + storage)
1. Map Main Course steps to SQLite table structure. If DB changes are needed, write a goose migration.
2. Create/update structs in `model/`.
3. Create `XStorage` interface and SQL implementation in `storage/`.

### Step 2: Business Logic (service)
1. Translate **Main Course** steps into the service method's main flow.
2. Translate **Exceptional Course** steps into checks and typed sentinel errors.
3. Define `XService` interface.

### Step 3: Testing
1. Write table-driven tests for the `service` layer.
2. Use lightweight struct mocks for `storage` directly inside `_test.go` (do not use mock generators).
3. Test the success path (Main Course) and **all** error branches (Exceptional Course).

### Step 4: Interface (views + handler)
1. Create `.templ` component with PicoCSS. Add HTMX attributes for data submission.
2. Write Echo handler: calls service, intercepts errors, renders either a success HTML fragment or a form with error.

**Atomicity:** Show the user the result after each step. Do not proceed to the next step without approval.

## 🎨 PicoCSS and HTMX Rules
- **PicoCSS:** Do not invent complex classes. Style via basic semantic tags (`<article>`, `<form>`, `<input>`, `<button>`). Use PicoCSS utility classes sparingly.
- **HTMX:** Handler returns a **targeted HTML fragment** for replacement (hx-swap="outerHTML", hx-target), not a full page reload.
- **Context:** If context is insufficient for precise integration, request the contents of adjacent files.

## 🚫 Hard Constraints (Strictly Enforced)
- **No ORM:** Only `database/sql` and raw SQL syntax.
- **No global variables:** Everything is passed via structs and Dependency Injection.
- **No reflection:** Forbidden to use `reflect` package, except in extreme unavoidable cases.
- **No `any` / `interface{}`:** Always declare concrete data types.
- **No `init()` functions:** All initialization must be explicit.
- **No `panic`:** Errors are values. Return them, do not crash the application.
- **No inheritance:** Go is not OOP, use composition.
- **Max 3 levels of nesting:** Avoid deep `if-else` conditions and loops (use early return).
- **Max 2 return values:** Strictly `(result, error)`.
- **Context first:** In all `service` and `storage` layer functions, `ctx context.Context` must be the first argument.
- **`(*T, error)` with nil = "not found"**, not an error
- **Build binaries into `bin/` folder**
- **Indexes and Column Selection**: When generating SQL queries, select only the needed columns (avoid `SELECT *`). If a query filters by a text field or foreign key, remind the developer to add an index in the goose migration file.
- This code is written for maximum autonomy and CPU efficiency. Avoid code bloat. The code must compile into a single binary capable of running without internet access on minimal hardware.

## 🔒 Safe database/sql practices (SQLite Rules)
- **Mandatory defer rows.Close()**: When calling `db.QueryContext` or `db.QueryRowContext`, close `rows` via `defer rows.Close()` STRICTLY AFTER checking `if err != nil`. Prevent connection leaks.
- **Check rows.Err()**: After the `for rows.Next()` loop, always check for errors via `if err := rows.Err(); err != nil`.
- **Safe transactions**: For operations modifying multiple tables or records, use `db.BeginTx(ctx, nil)`. Always call `defer tx.Rollback()`. Call `tx.Commit()` only at the very end after all steps succeed.
- **SQL injection prevention**: Never concatenate strings to build SQL queries. Always use placeholder arguments (`?` for SQLite).

## 🔄 Request Processing Flow (Data Flow)
Click/Form (htmx) ➔ handler (parsing) ➔ service (use case via interface) ➔ storage (SQL via interface) ➔ service ➔ handler (render templ component) ➔ htmx updates DOM on client.

## 🧪 Testing
- Test files `_test.go` are placed strictly next to the file being tested.
- Use exclusively **Table-Driven Tests**.
- Mocks are lightweight structs implementing the `storage` interface, defined inside the test file. No generators or external dependencies.

## 🔧 Build and Run
- `make build-bin` — build binary to `bin/` (templ generate + go build)
- `make check` — check code health and formatting (NOTE: use every time you change something to avoid breakage)
- `make dev` — live-reload via air

### Go codebase rules
1. After any code change, YOU MUST run `make check`.
2. If `make check` fails (exit code != 0):
   - Do not ignore console output.
   - Analyze the logs.
   - Fix syntax errors and linter warnings first, then fix failing tests.
   - Re-run `make check` after fixing.
3. Code is considered ready only when `make check` passes completely without warnings.
