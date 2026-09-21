# Frontend Type Safety

The browser code is plain JavaScript with no TypeScript compiler or runtime
schema library. API response shapes follow the backend contract and are
consumed defensively where practical.

Keep field names aligned with Go JSON tags (`path`, `dir`, `modified`, and
`preview`). Treat file names, paths, MIME values, and errors as untrusted:
use `esc` for HTML and `encodeURIComponent` (`enc`) for query parameters.

Do not use `eval` or unchecked HTML interpolation. If TypeScript is introduced,
add shared API types and runtime validation rather than silencing errors with
`any`.
