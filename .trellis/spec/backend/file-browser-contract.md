# File Browser Contract

## 1. Scope / Trigger

This project serves local files through an authenticated HTTP interface. Every request path and uploaded filename is untrusted input, and the service can be exposed through a reverse proxy.

## 2. Signatures

- `GET /api/list?path=<relative-path>` lists one directory.
- `GET /api/info?path=<relative-path>` returns file metadata and preview kind.
- `GET /api/preview?path=<relative-path>` returns a text preview.
- `GET /api/download?path=<relative-path>` downloads one file.
- `GET /api/media?path=<relative-path>` streams image, video, or PDF previews.
- `POST /api/upload?path=<relative-directory>` accepts a multipart `file` field.
- `POST /api/login` accepts JSON `{ "password": "..." }`; `POST /api/logout` removes the session.

## 3. Contracts

- Runtime requires `FILE_BROWSER_PASSWORD` or `-password`; `-root` defaults to `~/project`; `-listen` defaults to `127.0.0.1:8080`. A `root` value supplied through `config.json` must be absolute: JSON values do not perform shell-style `~` expansion.
- Successful login issues `file_browser_session` as an HttpOnly, SameSite=Lax cookie. Set `-cookie-secure` when the browser reaches the service over HTTPS.
- API failures use JSON `{ "error": "<message>" }`.
- Directory entries contain `name`, `path`, `dir`, `size`, `modified`, and `mime`; files are directories first, then name ascending.
- Text previews are limited to 5 MiB. `.txt` and `.md` files are supported when their content passes the UTF-8/control-character check; UTF-8 text is reported with `content`, `size`, `characters`, `lines`, `mime`, and `truncated`.
- Markdown previews use the `markdown` preview kind. `GET /api/preview` additionally returns `html`; the renderer must not enable raw HTML and must reject dangerous link protocols. The client may insert only this server-rendered value and must retain escaped source text for the raw view.

## 4. Validation & Error Matrix

| Condition | Response |
| --- | --- |
| Missing or invalid session | `401` for APIs; redirect to `/login.html` for pages |
| Path outside the configured root or unsafe link | `403` |
| Missing path | `404` |
| Directory passed to a file endpoint | `400` |
| Unsupported inline/text preview | `415` |
| Upload file name already exists | `409`; preserve existing bytes |
| Missing upload field or invalid name | `400` |
| Upload larger than the configured limit | `413` |

## 5. Good / Base / Bad Cases

- Good: open all local file-system paths via the persistent `os.Root` created for the configured root, then serve from the opened file handle.
- Base: return metadata and a download link for unrecognized binary files.
- Bad: validate a string path with `EvalSymlinks` and later pass that string to `os.Open`, `os.OpenFile`, or `http.ServeFile`; a replacement symlink can escape the root between those steps.

## 6. Tests Required

- Verify unauthenticated API and page behavior, plus login page static assets.
- Verify `..`, absolute paths, unsafe symlinks, and replacement of the configured root cannot read outside files.
- Verify an existing upload name returns `409` without changing the original file.
- Verify small UTF-8 text and configuration files preview, while binary data and files over 5 MiB do not expose full content.
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
