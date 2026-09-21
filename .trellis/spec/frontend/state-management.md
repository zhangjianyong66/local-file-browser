# Frontend State Management

There is no global state library or client-side server cache. Small page state
lives in `web/app.js`: current directory, selected file, selected name, and the
upload flag.

Server state is fetched on demand with `api`. `load` refreshes a directory and
clears selection; `show` fetches metadata and preview; successful uploads call
`load(current)`. The URL is used for markdown previews and download/media
endpoints. Do not duplicate API data or retain stale entries after mutations.
