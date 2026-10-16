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

HTML/HTM file clicks open `/html.html?path=<encoded-path>` in a new tab/window
using `noopener,noreferrer`. The preview shell provides a download link and loads
the server-issued preview URL in an iframe with `sandbox="allow-scripts"` and
`referrerpolicy="no-referrer"`; never grant `allow-same-origin` or insert raw
HTML into the shell. A null return from `window.open` with `noopener` does not
prove that a popup was blocked, so do not show an error based on that alone.
