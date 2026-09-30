# Frontend State Management

There is no global state library or client-side server cache. Small page state
lives in `web/app.js`: current directory, selected file, selected name, and the
upload flag.

Server state is fetched on demand with `api`. `load` refreshes a directory and
clears selection; `show` fetches metadata and preview; successful uploads call
`load(current)`. The URL is used for directory navigation, markdown previews, and download/media
endpoints. Do not duplicate API data or retain stale entries after mutations.

## Directory Routing Contract

### 1. Scope / Trigger

Apply this contract when changing directory navigation, asynchronous loading,
uploads that refresh listings, or authentication redirects.

### 2. Signatures

- `/` opens the root directory; `/?path=<relative-directory>` opens a directory.
- `/login.html?return=<encoded-directory-URL>` carries the login destination.
- `GET /api/list?path=<relative-directory>` returns the canonical directory path.
- `load(path, navigation)` uses `push` for user navigation and `replace` for
  initialization, history traversal, retry, and upload refresh.

### 3. Contracts

- Read `path` with URLSearchParams on startup and popstate; generate URLs with
  the same API so Chinese characters, spaces, `#`, and `%` round-trip correctly.
- A successful move to a different directory pushes history. Same-directory
  refresh and popstate replace the current entry without adding history.
- Track the requested target separately from the last successfully loaded
  directory. Retry and login return use the target.
- Directory changes clear file selection and preview; refreshing does not
  restore selected files or scroll position.
- Only the latest directory or preview request may update UI state or trigger
  a 401 redirect. Uploads stay bound to the directory where they started;
  completing an upload cannot navigate away from a newer directory.

### 4. Validation / Error Matrix

| Event | Required behavior |
| --- | --- |
| Directory fetch fails | Retain target URL; show retry and return-to-root actions; disable upload |
| Current API request returns 401 | Login with the target directory as return destination |
| Superseded request completes | Do not update state, messages, history, or login navigation |
| Login succeeds | Accept only same-origin `/` or `/index.html`; rebuild a `/` URL containing only path |
| Invalid login return address | Fall back to `/` |

### 5. Good / Base / Bad Cases

- Good: back/forward changes address, breadcrumbs, and listing together.
- Base: a root URL has no path query parameter.
- Bad: retry uses the previous successful directory after a target failed.

### 6. Tests Required

Run `node --test scripts/routing.test.cjs` for URL round-tripping, history,
failure actions, request ordering, upload navigation, 401 return, and unsafe
login destinations. Verify reload, back/forward, special directory names,
and re-login in a browser against a freshly built Go binary.

### 7. Wrong vs Correct

Wrong: initialize with `load()` regardless of URL or accept arbitrary login
return strings as redirect destinations.

Correct: initialize from URLSearchParams, use the target directory for retry,
and rebuild a validated same-origin directory URL before redirecting.
