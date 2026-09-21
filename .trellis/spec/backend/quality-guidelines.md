# Backend Quality Guidelines

Run `gofmt` on changed Go files, then `go test -race ./...`, `go vet ./...`,
and `go build ./...`. HTTP behavior should use `httptest.Server` and temporary
directories, and tests must close opened `os.Root` handles.

Preserve root-bound access through `server.rootFS`, authentication and status
codes from `file-browser-contract.md`, upload conflict behavior, preview
limits, and regression coverage for security-sensitive paths. Prefer standard
library APIs and the existing single-package structure.

Never use unchecked `os.Open`, `os.OpenFile`, `http.ServeFile`, or string-only
validation for user paths. Do not overwrite existing uploads or return secrets
in errors. Review path traversal, symlink races, new-route authentication,
resource cleanup, concurrent session access, size limits, and test coverage.
