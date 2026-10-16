# File Browser Contract

## 1. Scope / Trigger

This project serves local files through an authenticated HTTP interface. Every request path and uploaded filename is untrusted input, and the service can be exposed through a reverse proxy.

## 2. Signatures

- `GET /api/list?path=<relative-path>` lists one directory.
- `GET /api/info?path=<relative-path>` returns file metadata and preview kind.
- `GET /api/preview?path=<relative-path>` returns a text preview.
- `GET /api/download?path=<relative-path>` downloads one file.
- `GET /api/media?path=<relative-path>` streams image, video, or PDF previews.
- `GET /api/html-preview?path=<relative-html-path>` returns `{ "url": "/html-files/<preview-token>/<encoded-path>" }` for `.html` and `.htm` (case insensitive).
- `GET|HEAD /html-files/<preview-token>/<relative-path>` serves HTML preview documents and their relative resources.
- `POST /api/upload?path=<relative-directory>` accepts a multipart `file` field.
- `POST /api/login` accepts JSON `{ "password": "..." }`; `POST /api/logout` removes the session.

## 3. Contracts

- Runtime requires `FILE_BROWSER_PASSWORD` or `-password`; `-root` defaults to `~/project`; `-listen` defaults to `127.0.0.1:8080`. A `root` value supplied through `config.json` must be absolute: JSON values do not perform shell-style `~` expansion.
- Successful login issues `file_browser_session` as an HttpOnly, SameSite=Lax cookie. Set `-cookie-secure` when the browser reaches the service over HTTPS.
- Unauthenticated `/` and `/index.html` requests redirect to `/login.html?return=<encoded-directory-URL>`; the directory URL is rebuilt from only the `path` query parameter. Other protected static pages redirect to `/login.html`. Login accepts only a same-origin root or index return entry and rebuilds its directory URL; arbitrary destinations must never be accepted.
- API failures use JSON `{ "error": "<message>" }`.
- Directory entries contain `name`, `path`, `dir`, `size`, `modified`, and `mime`; files are directories first, then name ascending.
- Text previews are limited to 5 MiB. `.txt` and `.md` files are supported when their content passes the UTF-8/control-character check; UTF-8 text is reported with `content`, `size`, `characters`, `lines`, `mime`, and `truncated`.
- Markdown previews use the `markdown` preview kind. `GET /api/preview` additionally returns `html`; the renderer must not enable raw HTML and must reject dangerous link protocols. The client may insert only this server-rendered value and must retain escaped source text for the raw view.
- HTML previews use a separate read-only token bound to a valid login session. A token never authenticates management APIs; logout revokes it and session expiry invalidates it. Sandboxed documents have opaque origins, so relative resources must not rely on the SameSite login cookie.
- HTML resource URLs preserve path segments for relative CSS, images, and classic scripts. Open files through `os.Root`, reject directories, and ignore resource query parameters when resolving the file. Encode each URL path segment individually.
- Preview responses use `Cache-Control: no-store`, `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, and CSP `sandbox allow-scripts` without `allow-same-origin`. Restrict scripts/styles/images to local preview resources and inline/data content as appropriate; disable connections, frames, objects, forms, and base overrides. The wrapper iframe also retains `sandbox="allow-scripts"` across navigations.
- Management `/api/` requests with `Origin: null` or `Sec-Fetch-Site: cross-site` return `403`, including attempts to navigate from a sandboxed document. Ordinary external links to the browser UI remain supported.

## 4. Validation & Error Matrix

| Condition | Response |
| --- | --- |
| Missing or invalid session | `401` for APIs; directory pages redirect to login carrying their directory; other protected pages redirect to `/login.html` |
| Path outside the configured root or unsafe link | `403` |
| Missing path | `404` |
| Directory passed to a file endpoint | `400` |
| Unsupported inline/text preview | `415` |
| Non-HTML path passed to `/api/html-preview` | `415` |
| Missing, expired, or revoked preview token | `401` |
| Preview resource method other than GET/HEAD | `405` |
| Opaque-origin or cross-site management API access | `403` |
| Upload file name already exists | `409`; preserve existing bytes |
| Missing upload field or invalid name | `400` |
| Upload larger than the configured limit | `413` |

## 5. Good / Base / Bad Cases

- Good: open all local file-system paths via the persistent `os.Root` created for the configured root, then serve from the opened file handle.
- Base: return metadata and a download link for unrecognized binary files.
- Bad: validate a string path with `EvalSymlinks` and later pass that string to `os.Open`, `os.OpenFile`, or `http.ServeFile`; a replacement symlink can escape the root between those steps.
- Good: render HTML inside the persistent iframe sandbox and serve relative resources using an independently scoped read-only preview credential.
- Bad: expose the login session ID in a resource URL, grant HTML `allow-same-origin`, or insert raw HTML into the file browser DOM.

## 6. Tests Required

- Verify unauthenticated API and page behavior, plus login page static assets.
- Verify root/index login redirects preserve special-character directory paths and drop unrelated query parameters; frontend tests reject external or non-directory return destinations.
- Verify `..`, absolute paths, unsafe symlinks, and replacement of the configured root cannot read outside files.
- Verify an existing upload name returns `409` without changing the original file.
- Verify small UTF-8 text and configuration files preview, while binary data and files over 5 MiB do not expose full content.
- Verify HTML/HTM path encoding, resource MIME and relative paths, GET/HEAD behavior, query handling, root/symlink boundaries, isolation headers, cookie-free token access, token rejection by management APIs, logout and expiry revocation. Verify CSS/images/classic scripts and parent/storage isolation in a real browser when available.
- Run `go test -race ./...`, `go vet ./...`, and `go build ./...`.

## 7. Wrong vs Correct

### Wrong

```go
safe, _ := filepath.EvalSymlinks(candidate)
http.ServeFile(w, r, safe)
```

### Correct

```go
file, err := s.rootFS.Open(relativePath)
if err != nil { /* map to the API error */ }
defer file.Close()
http.ServeContent(w, r, name, info.ModTime(), file)
```
