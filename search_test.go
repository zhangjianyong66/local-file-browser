package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeSearchFile(t *testing.T, root, relative string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("search fixture"), 0644); err != nil {
		t.Fatal(err)
	}
}

func searchPaths(result searchResult) []string {
	paths := make([]string, 0, len(result.Entries))
	for _, e := range result.Entries {
		paths = append(paths, e.Path)
	}
	return paths
}

func TestSearchFileNamesAndPathSuffixes(t *testing.T) {
	s, root := testServer(t)
	suffix := "docs/design-drafts/wukong-v14-attack-grok-handoff-v3/prompt-zh.txt"
	for _, name := range []string{"A/" + suffix, "B/" + suffix, "C/my" + suffix, "other/prompt-zh.txt", "中文 空格#%/PROMPT-ZH.TXT", "docs/a.txt"} {
		writeSearchFile(t, root, name)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)
	cases := []struct {
		query, scope string
		want         []string
	}{
		{suffix, "", []string{"A/" + suffix, "B/" + suffix}},
		{"prompt-ZH", "A", []string{"A/" + suffix}},
		{" PROMPT-ZH ", "中文 空格#%", []string{"中文 空格#%/PROMPT-ZH.TXT"}},
		{"docs/a.txt", "", []string{"docs/a.txt"}},
		{"not-present", "", []string{}},
		{"Docs/a.txt", "", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.query+tc.scope, func(t *testing.T) {
			r := get(t, c, ts.URL+"/api/search?"+url.Values{"q": {tc.query}, "path": {tc.scope}}.Encode())
			defer r.Body.Close()
			if r.StatusCode != 200 {
				t.Fatalf("status %d", r.StatusCode)
			}
			var result searchResult
			if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(searchPaths(result), tc.want) {
				t.Fatalf("got %v want %v", searchPaths(result), tc.want)
			}
			if result.Truncated || result.Skipped != 0 || result.Query != strings.TrimSpace(tc.query) {
				t.Fatalf("unexpected result %+v", result)
			}
		})
	}
}

func TestSearchAuthenticationValidationAndScope(t *testing.T) {
	s, root := testServer(t)
	writeSearchFile(t, root, "note.txt")
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	r := get(t, c, ts.URL+"/api/search?q=note")
	r.Body.Close()
	if r.StatusCode != 401 {
		t.Fatalf("unauthenticated: %d", r.StatusCode)
	}
	login(t, c, ts.URL)
	for _, query := range []string{"", "  ", "../note.txt", "/note.txt", "docs//note.txt", "docs/./note.txt", "docs/../note.txt", "docs\\note.txt", "\x00", strings.Repeat("a", 4097)} {
		r := get(t, c, ts.URL+"/api/search?"+url.Values{"q": {query}}.Encode())
		r.Body.Close()
		if r.StatusCode != 400 {
			t.Errorf("query %q status %d", query, r.StatusCode)
		}
	}
	for _, tc := range []struct {
		scope  string
		status int
	}{{"../", 403}, {"/etc", 403}, {"missing", 404}, {"note.txt", 400}} {
		r := get(t, c, ts.URL+"/api/search?"+url.Values{"q": {"note"}, "path": {tc.scope}}.Encode())
		r.Body.Close()
		if r.StatusCode != tc.status {
			t.Errorf("scope %q: %d want %d", tc.scope, r.StatusCode, tc.status)
		}
	}
	r, err := c.Post(ts.URL+"/api/search?q=note", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 405 {
		t.Fatalf("method: %d", r.StatusCode)
	}
	req, _ := http.NewRequest("GET", ts.URL+"/api/search?q=note", nil)
	req.Header.Set("Origin", "null")
	r, err = c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 403 {
		t.Fatalf("opaque origin: %d", r.StatusCode)
	}
}

func TestSearchSymlinksAndPartialResults(t *testing.T) {
	s, root := testServer(t)
	writeSearchFile(t, root, "real/note.txt")
	outside := t.TempDir()
	writeSearchFile(t, outside, "secret-note.txt")
	for _, link := range []struct{ target, name string }{{"real/note.txt", "alias-note.txt"}, {".", "loop"}, {"real", "linked-dir"}, {outside, "outside"}, {"missing", "dangling-note.txt"}} {
		if err := os.Symlink(link.target, filepath.Join(root, link.name)); err != nil {
			t.Fatal(err)
		}
	}
	limits := searchLimits{results: 500, scanned: 100000, duration: time.Second}
	result, err := s.findFiles(context.Background(), ".", "note", limits)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"alias-note.txt", "real/note.txt"}; !reflect.DeepEqual(searchPaths(result), want) {
		t.Fatalf("got %+v", result)
	}
	if result.Skipped != 2 || result.Truncated {
		t.Fatalf("partial result %+v", result)
	}
	result, err = s.findFiles(context.Background(), "linked-dir", "note", limits)
	if err != nil || !reflect.DeepEqual(searchPaths(result), []string{"linked-dir/note.txt"}) {
		t.Fatalf("explicit internal alias: %+v %v", result, err)
	}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()
	c := clientFor(t, ts)
	login(t, c, ts.URL)
	r := get(t, c, ts.URL+"/api/search?q=secret&path=outside")
	defer r.Body.Close()
	if r.StatusCode != 403 {
		t.Fatalf("outside symlink: %d", r.StatusCode)
	}
}

func TestSearchBudgetsAndCancellation(t *testing.T) {
	s, root := testServer(t)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		writeSearchFile(t, root, name)
	}
	for _, tc := range []struct {
		limits searchLimits
		reason string
	}{
		{searchLimits{results: 1, scanned: 100, duration: time.Second}, "result_limit"},
		{searchLimits{results: 100, scanned: 1, duration: time.Second}, "scan_limit"},
		{searchLimits{results: 100, scanned: 100, duration: 0}, "time_limit"},
	} {
		result, err := s.findFiles(context.Background(), ".", ".txt", tc.limits)
		if err != nil || !result.Truncated || result.Reason != tc.reason {
			t.Fatalf("budget %+v: %+v %v", tc.limits, result, err)
		}
		if len(result.Entries) > tc.limits.results {
			t.Fatalf("result budget exceeded: %+v", result)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.findFiles(ctx, ".", "txt", searchLimits{results: 100, scanned: 100, duration: time.Second}); err != context.Canceled {
		t.Fatalf("cancel: %v", err)
	}
}

func TestSearchPathHintsReachProjectsBeforeLargeSubtrees(t *testing.T) {
	s, root := testServer(t)
	suffix := "docs/design-drafts/wukong-v14-attack-grok-handoff-v3/prompt-zh.txt"
	// Both subtrees exceed the entire test budget. Neither may hide sibling
	// project paths whose directory names are already supplied by the user.
	for _, project := range []string{"a-dependencies", "z-dependencies"} {
		for i := 0; i < 100; i++ {
			writeSearchFile(t, root, fmt.Sprintf("%s/package-%03d/noise.txt", project, i))
		}
	}
	for _, project := range []string{"project-a", "xiyou-idle"} {
		writeSearchFile(t, root, project+"/"+suffix)
	}
	writeSearchFile(t, root, "wrong/my"+suffix)
	result, err := s.findFiles(context.Background(), ".", suffix, searchLimits{results: 500, scanned: 50, duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"project-a/" + suffix, "xiyou-idle/" + suffix}
	if !reflect.DeepEqual(searchPaths(result), want) {
		t.Fatalf("got %+v, want %v", result, want)
	}
	if !result.Truncated || result.Reason != "scan_limit" {
		t.Fatalf("missing budget notice: %+v", result)
	}
}

func TestSearchPathProbeDeduplicatesAndDoesNotTraverseDirectoryLinks(t *testing.T) {
	s, root := testServer(t)
	suffix := "docs/deep/note.txt"
	writeSearchFile(t, root, "project/"+suffix)
	writeSearchFile(t, root, "real/deep/note.txt")
	if err := os.MkdirAll(filepath.Join(root, "linked-project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../real", filepath.Join(root, "linked-project/docs")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("note.txt", filepath.Join(root, "project/docs/deep/alias.txt")); err != nil {
		t.Fatal(err)
	}
	limits := searchLimits{results: 500, scanned: 10000, duration: time.Second}
	result, err := s.findFiles(context.Background(), ".", suffix, limits)
	if err != nil || !reflect.DeepEqual(searchPaths(result), []string{"project/" + suffix}) || result.Truncated {
		t.Fatalf("duplicate or directory link result: %+v %v", result, err)
	}
	result, err = s.findFiles(context.Background(), ".", "docs/deep/alias.txt", limits)
	if err != nil || !reflect.DeepEqual(searchPaths(result), []string{"project/docs/deep/alias.txt"}) {
		t.Fatalf("safe file link result: %+v %v", result, err)
	}
	result, err = s.findFiles(context.Background(), "project/docs", suffix, limits)
	if err != nil || !reflect.DeepEqual(searchPaths(result), []string{"project/" + suffix}) {
		t.Fatalf("scope nested within query: %+v %v", result, err)
	}
}

func TestSearchProbeScanBudgetAndCancellation(t *testing.T) {
	s, root := testServer(t)
	writeSearchFile(t, root, "docs/deep/note.txt")
	limits := searchLimits{results: 500, scanned: 2, duration: time.Second}
	result, err := s.findFiles(context.Background(), ".", "docs/deep/note.txt", limits)
	if err != nil || len(result.Entries) != 0 || !result.Truncated || result.Reason != "scan_limit" {
		t.Fatalf("probe exceeded scan budget: %+v %v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	limits.scanned = 1000
	result, err = s.findFiles(ctx, ".", "docs/deep/note.txt", limits)
	if err != context.Canceled || len(result.Entries) != 0 {
		t.Fatalf("canceled direct probe: %+v %v", result, err)
	}
	outside := t.TempDir()
	writeSearchFile(t, outside, "secret.txt")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "docs/deep/secret.txt")); err != nil {
		t.Fatal(err)
	}
	result, err = s.findFiles(context.Background(), ".", "docs/deep/secret.txt", limits)
	if err != nil || len(result.Entries) != 0 || result.Skipped == 0 || result.Truncated {
		t.Fatalf("outside file link must not become a probe result: %+v %v", result, err)
	}
	if err := os.Symlink("docs", filepath.Join(root, "chosen-scope")); err != nil {
		t.Fatal(err)
	}
	result, err = s.findFiles(context.Background(), "chosen-scope", "deep/note.txt", limits)
	if err != nil || !reflect.DeepEqual(searchPaths(result), []string{"chosen-scope/deep/note.txt"}) || result.Truncated {
		t.Fatalf("explicit alias scope lost or duplicated result: %+v %v", result, err)
	}
}
