# Logging Guidelines

The backend uses Go's standard `log` package. Startup logs the resolved root
and listen address; configuration and server-construction failures terminate
through `log.Fatal` in `main`.

Keep logs concise and operational. Log lifecycle/configuration failures, but
never passwords, cookie values, uploaded contents, or sensitive request data.
Expected client errors are communicated through HTTP responses rather than
per-request logs. If structured or leveled logging is introduced, update this
document and preserve the no-secrets rule.
