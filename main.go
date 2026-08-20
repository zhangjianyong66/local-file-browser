package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

const (
	maxPreviewBytes = 5 << 20
	maxUploadBytes  = 256 << 20
	sessionLifetime = 24 * time.Hour
)

//go:embed web/*
var webFS embed.FS

type config struct {
	root         string
	listen       string
	password     string
	cookieSecure bool
}

type fileConfig struct {
	Root         string `json:"root"`
	Listen       string `json:"listen"`
	Password     string `json:"password"`
	CookieSecure *bool  `json:"cookie_secure"`
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
}

type server struct {
	root         string // Evaluated physical root directory.
	rootFS       *os.Root
	password     string
	cookieSecure bool
	sessions     sessionStore
}

type entry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Dir      bool      `json:"dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	MIME     string    `json:"mime"`
}

type fileInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	MIME     string    `json:"mime"`
	Preview  string    `json:"preview"`
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}
	s, err := newServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("serving %s on http://%s", s.root, cfg.listen)
	log.Fatal(http.ListenAndServe(cfg.listen, s.routes()))
}

func parseConfig() (config, error) {
	return parseConfigArgs(os.Args[1:], os.Environ())
}

func parseConfigArgs(args, environ []string) (config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return config{}, fmt.Errorf("get home directory: %w", err)
	}
	env := environMap(environ)
	defaultRoot := filepath.Join(home, "project")
	configPath, explicitConfig := configPathFromArgs(args)
	if configPath == "" {
		exe, exeErr := os.Executable()
		if exeErr != nil {
			return config{}, fmt.Errorf("find executable for default config: %w", exeErr)
		}
		configPath = filepath.Join(filepath.Dir(exe), "config.json")
	}
	fileValues := fileConfig{}
	data, readErr := os.ReadFile(configPath)
	if readErr == nil {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&fileValues); err != nil {
			return config{}, fmt.Errorf("read config %s: %w", configPath, err)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) || explicitConfig {
		return config{}, fmt.Errorf("read config %s: %w", configPath, readErr)
	}

	rootValue := firstNonEmpty(fileValues.Root, defaultRoot)
	listenValue := firstNonEmpty(fileValues.Listen, "127.0.0.1:8080")
	passwordValue := fileValues.Password
	secureValue := false
	if fileValues.CookieSecure != nil {
		secureValue = *fileValues.CookieSecure
	}
	if v := env["FILE_BROWSER_ROOT"]; v != "" {
		rootValue = v
	}
	if v := env["FILE_BROWSER_LISTEN"]; v != "" {
		listenValue = v
	}
	if v, ok := env["FILE_BROWSER_PASSWORD"]; ok {
		passwordValue = v
	}
	if v, ok := env["FILE_BROWSER_COOKIE_SECURE"]; ok {
		secureValue, err = strconv.ParseBool(v)
		if err != nil {
			return config{}, fmt.Errorf("invalid FILE_BROWSER_COOKIE_SECURE: %w", err)
		}
	}

	set := flag.NewFlagSet("local-file-browser", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.String("config", configPath, "JSON configuration file")
	root := set.String("root", rootValue, "directory to browse")
	listen := set.String("listen", listenValue, "listen address")
	password := set.String("password", passwordValue, "login password (prefer FILE_BROWSER_PASSWORD)")
	secure := set.Bool("cookie-secure", secureValue, "set Secure on session cookies (requires HTTPS)")
	if err := set.Parse(args); err != nil {
		return config{}, err
	}
	if *password == "" {
		return config{}, errors.New("set FILE_BROWSER_PASSWORD or -password")
	}
	return config{root: *root, listen: *listen, password: *password, cookieSecure: *secure}, nil
}

func configPathFromArgs(args []string) (string, bool) {
	for i, arg := range args {
		if arg == "-config" || arg == "--config" {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", true
		}
		if strings.HasPrefix(arg, "-config=") {
			return strings.TrimPrefix(arg, "-config="), true
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config="), true
		}
	}
	return "", false
}

func environMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if ok {
			result[key] = val
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func newServer(cfg config) (*server, error) {
	root, err := filepath.EvalSymlinks(cfg.root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("root must be a directory")
	}
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}
	return &server{root: root, rootFS: rootFS, password: cfg.password, cookieSecure: cfg.cookieSecure, sessions: sessionStore{sessions: make(map[string]time.Time)}}, nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", s.login)
	mux.HandleFunc("/api/logout", s.logout)
	mux.HandleFunc("/api/list", s.list)
	mux.HandleFunc("/api/info", s.info)
	mux.HandleFunc("/api/preview", s.preview)
	mux.HandleFunc("/api/download", s.download)
	mux.HandleFunc("/api/media", s.media)
	mux.HandleFunc("/api/upload", s.upload)
	staticFiles, err := fs.Sub(webFS, "web")
	if err != nil {
		panic("embedded web assets are missing: " + err.Error())
	}
	mux.Handle("/", http.FileServer(http.FS(staticFiles)))
	return s.requireAuth(mux)
}

func (s *server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/login", "/login.html", "/style.css":
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("file_browser_session")
		if err == nil && s.validSession(cookie.Value) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		http.Redirect(w, r, "/login.html", http.StatusSeeOther)
	})
}

func (s *server) validSession(id string) bool {
	s.sessions.mu.Lock()
	defer s.sessions.mu.Unlock()
	expires, ok := s.sessions.sessions[id]
	if !ok || time.Now().After(expires) {
		delete(s.sessions.sessions, id)
		return false
	}
	return true
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil || body.Password != s.password {
		writeError(w, http.StatusUnauthorized, "invalid password")
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		writeError(w, 500, "create session")
		return
	}
	id := hex.EncodeToString(b)
	s.sessions.mu.Lock()
	s.sessions.sessions[id] = time.Now().Add(sessionLifetime)
	s.sessions.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "file_browser_session", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cookieSecure, MaxAge: int(sessionLifetime.Seconds())})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST required")
		return
	}
	if c, err := r.Cookie("file_browser_session"); err == nil {
		s.sessions.mu.Lock()
		delete(s.sessions.sessions, c.Value)
		s.sessions.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "file_browser_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cookieSecure})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// resolve returns an existing path relative to the Root handle.
func (s *server) resolve(relative string) (string, error) {
	rel, err := cleanRelative(relative)
	if err != nil {
		return "", err
	}
	if _, err := s.rootFS.Stat(rel); err != nil {
		return "", err
	}
	return rel, nil
}

func cleanRelative(value string) (string, error) {
	if value == "" || value == "." {
		return ".", nil
	}
	if strings.Contains(value, "\\") || path.IsAbs(value) {
		return "", fs.ErrPermission
	}
	parts := make([]string, 0)
	for _, part := range strings.Split(value, "/") {
		switch part {
		case "", ".":
		case "..":
			if len(parts) == 0 {
				return "", fs.ErrPermission
			}
			parts = parts[:len(parts)-1]
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return ".", nil
	}
	return strings.Join(parts, "/"), nil
}

func requestPath(r *http.Request) string { return r.URL.Query().Get("path") }

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET required")
		return
	}
	dirPath, err := s.resolve(requestPath(r))
	if err != nil {
		writePathError(w, err)
		return
	}
	dir, err := s.rootFS.Open(dirPath)
	if err != nil {
		writePathError(w, err)
		return
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		writePathError(w, err)
		return
	}
	result := make([]entry, 0, len(entries))
	for _, e := range entries {
		rel := path.Join(dirPath, e.Name())
		info, err := s.rootFS.Stat(rel)
		if err != nil {
			continue
		} // Never expose unsafe links.
		result = append(result, makeEntry(rel, info))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Dir != result[j].Dir {
			return result[i].Dir
		}
		return result[i].Name < result[j].Name
	})
	writeJSON(w, 200, map[string]any{"path": normalizedPath(requestPath(r)), "entries": result})
}

func makeEntry(relative string, info fs.FileInfo) entry {
	return entry{Name: info.Name(), Path: normalizedPath(relative), Dir: info.IsDir(), Size: info.Size(), Modified: info.ModTime(), MIME: mimeFor(info.Name(), info.IsDir())}
}
func normalizedPath(p string) string {
	p = strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
	if p == "." {
		return ""
	}
	return p
}
func mimeFor(name string, dir bool) string {
	if dir {
		return "inode/directory"
	}
	if v := mime.TypeByExtension(filepath.Ext(name)); v != "" {
		return v
	}
	return "application/octet-stream"
}

func (s *server) info(w http.ResponseWriter, r *http.Request) {
	file, info, ok := s.fileForRequest(w, r)
	if !ok {
		return
	}
	defer file.Close()
	if kind := mediaPreviewKind(info.Name()); kind != "" {
		writeJSON(w, 200, fileInfo{Path: normalizedPath(requestPath(r)), Name: info.Name(), Size: info.Size(), Modified: info.ModTime(), MIME: mimeFor(info.Name(), false), Preview: kind})
		return
	}
	data, truncated, err := readPreview(file)
	if err != nil {
		writeError(w, 500, "read preview")
		return
	}
	writeJSON(w, 200, fileInfo{Path: normalizedPath(requestPath(r)), Name: info.Name(), Size: info.Size(), Modified: info.ModTime(), MIME: mimeFor(info.Name(), false), Preview: previewKind(info.Name(), data, truncated)})
}

func previewKind(name string, data []byte, truncated bool) string {
	if kind := mediaPreviewKind(name); kind != "" {
		return kind
	}
	if strings.EqualFold(filepath.Ext(name), ".md") && isText(data, truncated) {
		return "markdown"
	}
	if explicitTextExtension(name) && isText(data, truncated) {
		return "text"
	}
	if isText(data, truncated) {
		return "text"
	}
	return "other"
}

func explicitTextExtension(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".md":
		return true
	default:
		return false
	}
}

func mediaPreviewKind(name string) string {
	m := mimeFor(name, false)
	if strings.HasPrefix(m, "image/") {
		return "image"
	}
	if strings.HasPrefix(m, "video/") {
		return "video"
	}
	if m == "application/pdf" {
		return "pdf"
	}
	return ""
}

func isText(data []byte, truncated bool) bool {
	if truncated && !utf8.Valid(data) {
		for range utf8.UTFMax - 1 {
			data = data[:len(data)-1]
			if utf8.Valid(data) {
				break
			}
		}
	}
	if bytes.ContainsRune(data, 0) || !utf8.Valid(data) {
		return false
	}
	for _, b := range data {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' && b != '\f' {
			return false
		}
	}
	return true
}

func readPreview(file *os.File) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(file, maxPreviewBytes+1))
	if err != nil {
		return nil, false, err
	}
	return data, len(data) > maxPreviewBytes, nil
}

func (s *server) fileForRequest(w http.ResponseWriter, r *http.Request) (*os.File, fs.FileInfo, bool) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "GET required")
		return nil, nil, false
	}
	filePath, err := s.resolve(requestPath(r))
	if err != nil {
		writePathError(w, err)
		return nil, nil, false
	}
	file, err := s.rootFS.Open(filePath)
	if err != nil {
		writePathError(w, err)
		return nil, nil, false
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		writePathError(w, err)
		return nil, nil, false
	}
	if info.IsDir() {
		file.Close()
		writeError(w, 400, "directories are not files")
		return nil, nil, false
	}
	return file, info, true
}

func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	file, info, ok := s.fileForRequest(w, r)
	if !ok {
		return
	}
	defer file.Close()
	data, truncated, err := readPreview(file)
	if err != nil {
		writeError(w, 500, "read preview")
		return
	}
	kind := previewKind(info.Name(), data, truncated)
	if kind != "text" && kind != "markdown" {
		writeError(w, 415, "text preview is unavailable for this file")
		return
	}
	if truncated {
		writeJSON(w, 200, map[string]any{"truncated": true, "limit": maxPreviewBytes})
		return
	}
	text := string(data)
	if kind == "markdown" {
		var rendered bytes.Buffer
		if err := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList)).Convert(data, &rendered); err != nil {
			writeError(w, 500, "render markdown")
			return
		}
		writeJSON(w, 200, map[string]any{"content": text, "html": rendered.String(), "size": len(data), "characters": utf8.RuneCountInString(text), "lines": strings.Count(text, "\n"), "mime": mimeFor(info.Name(), false), "truncated": false})
		return
	}
	writeJSON(w, 200, map[string]any{"content": text, "size": len(data), "characters": utf8.RuneCountInString(text), "lines": strings.Count(text, "\n"), "mime": mimeFor(info.Name(), false), "truncated": false})
}

func (s *server) download(w http.ResponseWriter, r *http.Request) {
	file, info, ok := s.fileForRequest(w, r)
	if ok {
		defer file.Close()
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", info.Name()))
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}
func (s *server) media(w http.ResponseWriter, r *http.Request) {
	file, info, ok := s.fileForRequest(w, r)
	if !ok {
		return
	}
	defer file.Close()
	if mediaPreviewKind(info.Name()) == "" {
		writeError(w, 415, "inline preview unavailable")
		return
	}
	w.Header().Set("Content-Type", mimeFor(info.Name(), false))
	w.Header().Set("Content-Disposition", "inline")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (s *server) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "POST required")
		return
	}
	dir, err := s.resolve(requestPath(r))
	if err != nil {
		writePathError(w, err)
		return
	}
	if info, err := s.rootFS.Stat(dir); err != nil || !info.IsDir() {
		writeError(w, 400, "upload destination must be a directory")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, 413, "upload exceeds size limit")
		return
	}
	source, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "missing file")
		return
	}
	defer source.Close()
	name := filepath.Base(header.Filename)
	if name == "." || name == "" || strings.Contains(name, string(os.PathSeparator)) {
		writeError(w, 400, "invalid filename")
		return
	}
	target := path.Join(dir, name)
	dest, err := s.rootFS.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, fs.ErrExist) {
		writeError(w, 409, "a file with that name already exists")
		return
	}
	if err != nil {
		writeError(w, 500, "create upload: "+err.Error())
		return
	}
	_, copyErr := io.Copy(dest, source)
	closeErr := dest.Close()
	if copyErr != nil || closeErr != nil {
		_ = s.rootFS.Remove(target)
		writeError(w, 500, "write upload")
		return
	}
	writeJSON(w, 201, map[string]any{"name": name, "path": path.Join(normalizedPath(requestPath(r)), name)})
}

func writePathError(w http.ResponseWriter, err error) {
	if errors.Is(err, fs.ErrPermission) || strings.Contains(err.Error(), "path escapes from parent") {
		writeError(w, 403, "path is outside the browse root")
		return
	}
	if errors.Is(err, fs.ErrNotExist) {
		writeError(w, 404, "path not found")
		return
	}
	writeError(w, 400, "invalid path")
}
func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
