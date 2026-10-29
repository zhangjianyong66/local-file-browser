package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type searchLimits struct {
	results  int
	scanned  int
	duration time.Duration
}

type searchResult struct {
	Path      string  `json:"path"`
	Query     string  `json:"query"`
	Entries   []entry `json:"entries"`
	Truncated bool    `json:"truncated"`
	Reason    string  `json:"reason"`
	Skipped   int     `json:"skipped"`
}

func validSearchQuery(query string) bool {
	if query == "" || len(query) > 4096 || !utf8.ValidString(query) || strings.ContainsAny(query, "\\\x00*?[]") {
		return false
	}
	for _, part := range strings.Split(query, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if !validSearchQuery(query) {
		writeError(w, http.StatusBadRequest, "请输入有效文件名或相对路径（不支持通配符和父目录跳转）")
		return
	}
	dirPath, err := s.resolve(requestPath(r))
	if err != nil {
		writePathError(w, err)
		return
	}
	info, err := s.rootFS.Stat(dirPath)
	if err != nil {
		writePathError(w, err)
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "search scope must be a directory")
		return
	}
	result, err := s.findFiles(r.Context(), dirPath, query, searchLimits{results: 500, scanned: 100000, duration: 5 * time.Second})
	if err != nil {
		if r.Context().Err() == nil {
			writePathError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) findFiles(ctx context.Context, scope, query string, limits searchLimits) (searchResult, error) {
	result := searchResult{Path: normalizedPath(scope), Query: query, Entries: make([]entry, 0)}
	deadline := time.Now().Add(limits.duration)
	scanned := 0
	stop := func() bool {
		switch {
		case len(result.Entries) >= limits.results:
			result.Reason = "result_limit"
		case scanned >= limits.scanned:
			result.Reason = "scan_limit"
		case !time.Now().Before(deadline):
			result.Reason = "time_limit"
		default:
			return false
		}
		result.Truncated = true
		return true
	}
	lowerQuery := strings.ToLower(query)
	pathQuery := strings.Contains(query, "/")
	seen := make(map[string]bool)
	appendMatch := func(relative string, info fs.FileInfo) {
		if seen[relative] {
			return
		}
		seen[relative] = true
		e := makeEntry(relative, info)
		e.Name = path.Base(relative)
		result.Entries = append(result.Entries, e)
	}
	// A supplied path is a strong hint: check it beneath each discovered
	// directory before enumerating its potentially huge contents. Each segment
	// is checked through os.Root, and directory symlinks remain untraversed.
	probePath := func(base string) error {
		if !pathQuery {
			return nil
		}
		relative := base
		parts := strings.Split(query, "/")
		for i, part := range parts {
			if err := ctx.Err(); err != nil {
				return err
			}
			if stop() {
				return nil
			}
			scanned++
			relative = path.Join(relative, part)
			info, err := s.rootFS.Lstat(relative)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					result.Skipped++
				}
				return nil
			}
			if i < len(parts)-1 {
				if !info.IsDir() {
					return nil
				}
				continue
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				info, err = s.rootFS.Stat(relative)
				if err != nil {
					result.Skipped++
					return nil
				}
			}
			if info.Mode().IsRegular() {
				appendMatch(relative, info)
			}
		}
		return nil
	}
	if err := probePath(scope); err != nil {
		return result, err
	}
	queue := []string{scope}
	// Keep one directory handle open at a time, even in deeply nested trees.
	// Breadth-first order reaches sibling projects before dependency subtrees.
	for len(queue) > 0 && !stop() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		dirPath := queue[0]
		queue[0] = ""
		queue = queue[1:]
		dir, err := s.rootFS.Open(dirPath)
		if err != nil {
			if dirPath == scope {
				return result, err
			}
			result.Skipped++
			continue
		}
		for !stop() {
			if err := ctx.Err(); err != nil {
				dir.Close()
				return result, err
			}
			batch, readErr := dir.ReadDir(128)
			for _, child := range batch {
				if err := ctx.Err(); err != nil {
					dir.Close()
					return result, err
				}
				if stop() {
					break
				}
				scanned++
				rel := path.Join(dirPath, child.Name())
				info, err := s.rootFS.Lstat(rel)
				if err != nil {
					result.Skipped++
					continue
				}
				isLink := info.Mode()&fs.ModeSymlink != 0
				if isLink {
					info, err = s.rootFS.Stat(rel)
					if err != nil {
						result.Skipped++
						continue
					}
				}
				if info.IsDir() {
					if !isLink {
						queue = append(queue, rel)
						if err := probePath(rel); err != nil {
							dir.Close()
							return result, err
						}
					}
					continue
				}
				if !info.Mode().IsRegular() {
					continue
				}
				matches := strings.Contains(strings.ToLower(child.Name()), lowerQuery)
				if pathQuery {
					matches = rel == query || strings.HasSuffix(rel, "/"+query)
				}
				if matches {
					appendMatch(rel, info)
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					result.Skipped++
				}
				break
			}
		}
		if err := dir.Close(); err != nil {
			result.Skipped++
		}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	// A final batch can hit a budget just before EOF; report conservatively.
	stop()
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].Path < result.Entries[j].Path })
	return result, nil
}
