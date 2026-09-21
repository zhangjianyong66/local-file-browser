# Error Handling

Handlers map failures to an HTTP status and `{"error":"message"}` through
`writeError`. API clients should rely on the status code and this field, not on
HTML error pages. Use wrapped errors (`fmt.Errorf("context: %w", err)`) while
propagating failures.

Classify missing paths, invalid input, unsafe paths, unsupported previews,
conflicts, and size limits according to `file-browser-contract.md`.
Authentication failures are `401`; page requests redirect to `/login.html`.

Do not expose local paths, passwords, session tokens, stack traces, or raw
filesystem errors to clients. Do not ignore errors from file opens, multipart
parsing, response writes, or session operations. Startup failures are fatal in
`main`; request failures should produce a response and return.
