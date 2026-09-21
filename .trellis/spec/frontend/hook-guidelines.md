# Frontend Hook Guidelines

No React or hook library is used, so custom hooks do not apply. Stateful browser
behavior is plain JavaScript module-level state in `web/app.js`: `current`,
`selected`, `selectedName`, and `isUploading`.

Use `async` functions for API workflows and centralize fetch behavior in `api`.
That helper handles expired sessions, JSON parsing, and server error messages.
Reset loading and disabled state in `finally`, as the upload and logout
handlers do. Do not add a hook-like abstraction without documenting its
lifecycle and dependency here.
