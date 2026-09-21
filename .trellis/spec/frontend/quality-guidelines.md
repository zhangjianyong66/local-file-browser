# Frontend Quality Guidelines

Keep the UI keyboard accessible and usable on narrow screens. Preserve the
skip link, semantic headings, labels, focus-visible styles, live status region,
reduced-motion rule, and meaningful media `alt`/`title` attributes. Escape all
user/server strings and encode file paths.

When API behavior changes, cover it with Go HTTP tests. For UI changes, check
login, directory load, file preview, upload success/conflict, logout, errors,
and mobile layout in a browser.

Do not add a framework or build step for isolated changes, inject raw file
content into `innerHTML`, remove authentication redirects, or hide failures.
Shared styles belong in `web/style.css`; the only intentional HTML insertion
of server content is sanitized markdown from the preview endpoint.
