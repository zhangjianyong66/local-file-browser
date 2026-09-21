# Frontend Component Guidelines

There is no component framework. Reusable UI is semantic HTML plus small
rendering functions in `web/app.js`, such as `renderList`, `breadcrumbs`,
`show`, and `setMessage`.

Use the existing `q` selector helper, keep DOM updates explicit, and escape
untrusted values with `esc` before interpolating HTML. Server-rendered markdown
is the deliberate exception: it is sanitized on the server and inserted only
in the markdown preview container.

Prefer native controls, labels, headings, links, and buttons. Preserve visible
focus states, `aria-live` messaging, skip navigation, useful media attributes,
and keyboard-operable actions. Keep styles in `web/style.css`.
