# File Search Contract

## 1. Scope / Trigger

Apply when changing file search, traversal budgets or search response rendering.

## 2. Signatures

`GET /api/search?q=<query>&path=<relative-directory>` returns `{path, query, entries, truncated, reason, skipped}`. Entries use the existing listing schema and root-relative paths. Empty path searches the entire library.

## 3. Contracts

- Trim query whitespace; require valid UTF-8 and 1–4096 bytes. Reject backslashes, NUL, wildcard syntax, empty path segments, dot/parent segments and absolute paths.
- Without a slash, match a case-insensitive substring of the filename. With slashes, match the exact case-sensitive relative path or a complete segment-boundary suffix. Never fall back to basename matching.
- Resolve scope and access every file through server.rootFS; scope must be a directory. Return regular files and safe file symlinks; do not recurse directory symlinks. Explicitly selected safe internal directory aliases may be scopes.
- Enforce 500 matches, 100000 scanned entries/probes and a 5-second traversal budget. Check cancellation, batch directory reads, close handles and deduplicate paths. Blocking filesystem calls may exceed the time budget.
- Use breadth-first traversal. Probe complete query paths under discovered ordinary directories, validating each segment and charging probes to the same budget. This checks shallow project paths before deep dependency trees consume the budget.
- Sort by full relative path. truncated indicates an exhausted budget; reason is result_limit, scan_limit or time_limit, otherwise empty. skipped counts inaccessible entries/probes and may count repeated access attempts.

## 4. Validation & Error Matrix

| Condition | Behavior |
| --- | --- |
| Invalid query or non-directory scope | 400 JSON error |
| Non-GET | 405 |
| Missing session / cross-site API access | Existing 401 / 403 rules |
| Missing / outside-root scope | Existing 404 / 403 rules |
| Unreadable descendant | Keep results and increment skipped |
| Budget reached | 200 with partial matches and truncated/reason |
| Canceled request | Stop traversal and close handles |

## 5. Good / Base / Bad Cases

- Good: docs/a.txt finds project-a/docs/a.txt and project-b/docs/a.txt.
- Base: A.TXT finds filename a.txt within scope.
- Bad: mydocs/a.txt matches the docs/a.txt query, or an unrelated a.txt appears for a path query.

## 6. Tests Required

Run go test -race ./..., go vet ./... and go build ./.... Cover filename/path matching, segment boundaries, special characters, multi-project matches with small budgets and distracting deep trees, scope, authentication, symlinks, cancellation, budgets, deduplication and partial results.

## 7. Wrong vs Correct

Wrong: reduce path queries to basenames, traverse without bounds, or open absolute paths after string-only validation.

Correct: preserve path-segment suffixes, use bounded breadth-first traversal and safe probes, and keep every filesystem operation within os.Root.
