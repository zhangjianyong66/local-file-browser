# Backend Directory Structure

## Current layout

```text
main.go       HTTP server, configuration, handlers, and file-system access
main_test.go  HTTP integration and configuration tests
web/          embedded static frontend assets
deploy/       systemd, nginx, and FRP deployment examples
scripts/      local process-management scripts
```

The backend is intentionally a single `main` package. Keep configuration near
`parseConfigArgs`, server construction near `newServer`, route registration in
`(*server).routes`, and one handler per API operation. File-system safety
helpers stay close to the handlers that use them. Do not add `routes/`,
`services/`, or `models/` packages without a real package boundary.

Use idiomatic Go names and explicit JSON tags, as in `entry` and `fileInfo` in
`main.go`. Add behavior tests in `main_test.go`; its authentication, symlink,
root-replacement, upload, and preview tests are the reference patterns.
