# Database Guidelines

This service has no database, ORM, migrations, or persistent application data.
The configured file system is the application data source; sessions are held
in-memory in `sessionStore` and disappear on process restart.

Do not add a database abstraction for listings, sessions, or uploads. If
persistence is introduced, document its technology and migration workflow here
before adding repository code.

User-controlled paths are relative to the configured root and must be opened
through the persistent `os.Root` (`server.rootFS`). Never validate a path and
then reopen an unchecked string path. See `file-browser-contract.md` and the
symlink/root-replacement tests in `main_test.go`.
