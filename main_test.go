package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testServer(t *testing.T) (*server, string) {
	t.Helper()
	root := t.TempDir()
	s, err := newServer(config{root: root, password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.rootFS.Close() })
	return s, root
}

func TestParseConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"root":"from-file","listen":"127.0.0.1:9000","password":"file-pass","cookie_secure":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseConfigArgs([]string{"-config", configPath, "-root", "from-flag", "-cookie-secure"}, []string{"FILE_BROWSER_PASSWORD=env-pass", "FILE_BROWSER_LISTEN=127.0.0.1:9100"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.root != "from-flag" || cfg.listen != "127.0.0.1:9100" || cfg.password != "env-pass" || !cfg.cookieSecure {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestParseConfigRejectsInvalidJSONAndBoolean(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"root":`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfigArgs([]string{"-config", configPath}, nil); err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if err := os.WriteFile(configPath, []byte(`{"password":"pass"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfigArgs([]string{"-config", configPath}, []string{"FILE_BROWSER_COOKIE_SECURE=maybe"}); err == nil {
		t.Fatal("expected invalid boolean error")
	}
}
func clientFor(t *testing.T, ts *httptest.Server) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(r *http.Request, v []*http.Request) error { return http.ErrUseLastResponse }}
}
func login(t *testing.T, c *http.Client, base string) {
	t.Helper()
	r, err := c.Post(base+"/api/login", "application/json", strings.NewReader(`{"password":"secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("login status %d", r.StatusCode)
	}
}
func get(t *testing.T, c *http.Client, u string) *http.Response {
	t.Helper()
	r, e := c.Get(u)
	if e != nil {
		t.Fatal(e)
	}
	return r
}

func TestAuthenticationAndDirectoryListing(t *testing.T) {
	s, root := testServer(t)
	if err := os.Mkdir(filepath.Join(root, "adir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "z.txt"), []byte("z"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden"), []byte("h"), 0644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	r := get(t, c, ts.URL+"/api/list")
	if r.StatusCode != 401 {
		t.Fatalf("got %d", r.StatusCode)
	}
	r.Body.Close()
	login(t, c, ts.URL)
	r = get(t, c, ts.URL+"/api/list")
	defer r.Body.Close()
	var body struct {
		Entries []entry `json:"entries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 3 || body.Entries[0].Name != "adir" || body.Entries[1].Name != ".hidden" {
		t.Fatalf("unexpected entries %#v", body.Entries)
	}
}

func TestStaticAppIsProtectedAndServedAfterLogin(t *testing.T) {
	s, _ := testServer(t)
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	r := get(t, c, ts.URL+"/")
	if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/login.html" {
		t.Fatalf("unexpected unauthenticated response: %d %q", r.StatusCode, r.Header.Get("Location"))
	}
	r.Body.Close()
	r = get(t, c, ts.URL+"/login.html")
	if r.StatusCode != http.StatusOK {
		t.Fatalf("login page status %d", r.StatusCode)
	}
	r.Body.Close()
	r = get(t, c, ts.URL+"/style.css")
	if r.StatusCode != http.StatusOK {
		t.Fatalf("login stylesheet status %d", r.StatusCode)
	}
	r.Body.Close()
	r = get(t, c, ts.URL+"/app.js")
	if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/login.html" {
		t.Fatalf("unexpected unauthenticated app script response: %d %q", r.StatusCode, r.Header.Get("Location"))
	}
	r.Body.Close()
	login(t, c, ts.URL)
	r = get(t, c, ts.URL+"/")
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil || r.StatusCode != http.StatusOK || !strings.Contains(string(body), "本地文件浏览器") || !strings.Contains(string(body), `href="/style.css"`) || !strings.Contains(string(body), `src="/app.js"`) {
		t.Fatalf("static app response: %d %v %q", r.StatusCode, err, body)
	}
	r = get(t, c, ts.URL+"/app.js")
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("authenticated app script status %d", r.StatusCode)
	}
	r = get(t, c, ts.URL+"/markdown.html")
	defer r.Body.Close()
	body, err = io.ReadAll(r.Body)
	if err != nil || r.StatusCode != http.StatusOK || !strings.Contains(string(body), "Markdown 预览") {
		t.Fatalf("markdown reader response: %d %v", r.StatusCode, err)
	}
}
func TestRejectsOutsideSymlink(t *testing.T) {
	s, root := testServer(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("no"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)
	r := get(t, c, ts.URL+"/api/download?path=escape/secret")
	defer r.Body.Close()
	if r.StatusCode != 403 {
		t.Fatalf("got %d", r.StatusCode)
	}
}

func TestRejectsOutsideSymlinkAsUploadDestination(t *testing.T) {
	s, root := testServer(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	p, err := mw.CreateFormFile("file", "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write([]byte("content")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := c.Post(ts.URL+"/api/upload?path=escape", mw.FormDataContentType(), &b)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d", r.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("upload escaped root: %v", err)
	}
}

func TestRootHandleResistsRootReplacement(t *testing.T) {
	s, root := testServer(t)
	if err := os.WriteFile(filepath.Join(root, "inside.txt"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	movedRoot := filepath.Join(filepath.Dir(root), "moved-root")
	if err := os.Rename(root, movedRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)
	r := get(t, c, ts.URL+"/api/download?path=inside.txt")
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil || r.StatusCode != http.StatusOK || string(body) != "inside" {
		t.Fatalf("inside response: %d %v %q", r.StatusCode, err, body)
	}
	r = get(t, c, ts.URL+"/api/download?path=secret.txt")
	defer r.Body.Close()
	if r.StatusCode != http.StatusNotFound {
		t.Fatalf("outside response: %d", r.StatusCode)
	}
}

func TestUploadConflictDoesNotOverwrite(t *testing.T) {
	s, root := testServer(t)
	if err := os.WriteFile(filepath.Join(root, "present.txt"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	p, _ := mw.CreateFormFile("file", "present.txt")
	_, _ = p.Write([]byte("replacement"))
	_ = mw.Close()
	r, err := c.Post(ts.URL+"/api/upload", mw.FormDataContentType(), &b)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 409 {
		t.Fatalf("got %d", r.StatusCode)
	}
	data, err := os.ReadFile(filepath.Join(root, "present.txt"))
	if err != nil || string(data) != "original" {
		t.Fatalf("overwrite: %q %v", data, err)
	}
}
func TestTextPreviewAndLargeFile(t *testing.T) {
	s, root := testServer(t)
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Project\n\n说明内容\n\n| 名称 | 状态 |\n| --- | --- |\n| 浏览器 | 正常 |\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unsafe.md"), []byte("# Safe\n\n<script>alert(1)</script>\n[bad](javascript:alert(1))\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README"), []byte("no extension\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.conf"), []byte("name=文件浏览器\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "binary.txt"), []byte{'a', 0, 'b'}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "control.dat"), []byte{'a', 1, 'b'}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.txt"), bytes.Repeat([]byte("x"), maxPreviewBytes+1), 0644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)
	r := get(t, c, ts.URL+"/api/preview?path=note.txt")
	defer r.Body.Close()
	var p struct {
		Characters, Lines int
		Truncated         bool
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.Characters != 8 || p.Lines != 2 || p.Truncated {
		t.Fatalf("preview %#v", p)
	}
	r = get(t, c, ts.URL+"/api/info?path=README.md")
	var info fileInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if info.Preview != "markdown" {
		t.Fatalf("markdown preview kind: %#v", info)
	}
	r = get(t, c, ts.URL+"/api/preview?path=README.md")
	var markdown struct {
		Content   string
		HTML      string `json:"html"`
		Truncated bool
	}
	if err := json.NewDecoder(r.Body).Decode(&markdown); err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if markdown.Truncated || !strings.Contains(markdown.Content, "# Project") || !strings.Contains(markdown.HTML, "<h1") || !strings.Contains(markdown.HTML, "<table>") {
		t.Fatalf("markdown preview: %#v", markdown)
	}
	r = get(t, c, ts.URL+"/api/preview?path=unsafe.md")
	var unsafe struct {
		HTML string `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&unsafe); err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if strings.Contains(unsafe.HTML, "<script") || strings.Contains(unsafe.HTML, "javascript:") {
		t.Fatalf("unsafe markdown rendered: %s", unsafe.HTML)
	}
	for _, name := range []string{"README", "settings.conf"} {
		r = get(t, c, ts.URL+"/api/info?path="+name)
		var info fileInfo
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			r.Body.Close()
			t.Fatal(err)
		}
		r.Body.Close()
		if info.Preview != "text" {
			t.Fatalf("%s preview kind %q", name, info.Preview)
		}
		r = get(t, c, ts.URL+"/api/preview?path="+name)
		if r.StatusCode != http.StatusOK {
			r.Body.Close()
			t.Fatalf("%s preview status %d", name, r.StatusCode)
		}
		r.Body.Close()
	}
	for _, name := range []string{"binary.txt", "control.dat"} {
		r = get(t, c, ts.URL+"/api/info?path="+name)
		var binaryInfo fileInfo
		if err := json.NewDecoder(r.Body).Decode(&binaryInfo); err != nil {
			r.Body.Close()
			t.Fatal(err)
		}
		r.Body.Close()
		if binaryInfo.Preview != "other" {
			t.Fatalf("%s preview kind %q", name, binaryInfo.Preview)
		}
		r = get(t, c, ts.URL+"/api/preview?path="+name)
		if r.StatusCode != http.StatusUnsupportedMediaType {
			r.Body.Close()
			t.Fatalf("%s preview status %d", name, r.StatusCode)
		}
		r.Body.Close()
	}
	r = get(t, c, ts.URL+"/api/preview?path=large.txt")
	defer r.Body.Close()
	var l struct{ Truncated bool }
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		t.Fatal(err)
	}
	if !l.Truncated {
		t.Fatal("large file was not truncated")
	}
}
func TestCleanRelative(t *testing.T) {
	for _, p := range []string{"../x", "/tmp/x", "a\\b"} {
		if _, err := cleanRelative(p); err == nil {
			t.Fatalf("accepted %q", p)
		}
	}
	if got, err := cleanRelative("a/../b"); err != nil || got != "b" {
		t.Fatalf("got %q %v", got, err)
	}
}
