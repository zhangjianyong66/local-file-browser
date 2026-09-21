# Frontend Directory Structure

This is a framework-free embedded frontend:

```text
web/index.html       authenticated file browser shell
web/login.html       login page
web/markdown.html    markdown reader page
web/app.js           browser state, API calls, and DOM rendering
web/style.css        shared responsive styles and design tokens
web/overflow-fix.css narrow overflow adjustments
```

There are no component, page, hook, or asset subdirectories. Keep page markup
in its HTML file, shared interaction in `web/app.js`, and visual rules in
`web/style.css`. New standalone pages belong under `web/` and are embedded by
the `//go:embed web/*` declaration. Use lowercase kebab-case filenames.
