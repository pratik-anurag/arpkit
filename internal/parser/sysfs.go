package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const defaultMaxFileBytes int64 = 8 << 20

type ErrFileTooLarge struct {
	Path  string
	Limit int64
}

func (e *ErrFileTooLarge) Error() string {
	return fmt.Sprintf("read %s: file exceeds %d bytes", e.Path, e.Limit)
}

type ReadIssue struct {
	Path  string
	Limit int64
}

// Reader is a small cached filesystem reader used for sysfs/procfs parsing.
type Reader struct {
	Root         string
	MaxFileBytes int64

	mu     sync.Mutex
	cache  map[string][]byte
	issues map[string]ReadIssue
}

func NewReader(root string) *Reader {
	if root == "" {
		root = "/"
	}
	return &Reader{
		Root:         root,
		MaxFileBytes: defaultMaxFileBytes,
		cache:        make(map[string][]byte, 512),
		issues:       make(map[string]ReadIssue, 8),
	}
}

func (r *Reader) ReadFile(path string) (string, error) {
	abs := r.resolve(path)
	logical := cleanLogicalPath(path)

	r.mu.Lock()
	if b, ok := r.cache[abs]; ok {
		r.mu.Unlock()
		return strings.TrimSpace(string(b)), nil
	}
	r.mu.Unlock()

	b, err := readFileBounded(abs, logical, r.maxFileBytes())
	if err != nil {
		var tooLarge *ErrFileTooLarge
		if errors.As(err, &tooLarge) {
			r.recordIssue(ReadIssue{Path: tooLarge.Path, Limit: tooLarge.Limit})
		}
		return "", err
	}

	r.mu.Lock()
	r.cache[abs] = b
	r.mu.Unlock()

	return strings.TrimSpace(string(b)), nil
}

func (r *Reader) ReadInt(path string) (int, error) {
	v, err := r.ReadFile(path)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("parse int %s: %w", path, err)
	}
	return n, nil
}

func (r *Reader) ReadUint64(path string) (uint64, error) {
	v, err := r.ReadFile(path)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse uint %s: %w", path, err)
	}
	return n, nil
}

func (r *Reader) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(r.resolve(path))
}

func (r *Reader) Exists(path string) bool {
	_, err := os.Stat(r.resolve(path))
	return err == nil
}

func (r *Reader) Issues() []ReadIssue {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]ReadIssue, 0, len(r.issues))
	for _, issue := range r.issues {
		out = append(out, issue)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func (r *Reader) resolve(path string) string {
	clean := cleanLogicalPath(path)
	clean = strings.TrimPrefix(clean, string(filepath.Separator))
	if r.Root == "/" {
		if clean == "" {
			return string(filepath.Separator)
		}
		return string(filepath.Separator) + clean
	}
	return filepath.Join(r.Root, clean)
}

func (r *Reader) maxFileBytes() int64 {
	if r == nil || r.MaxFileBytes <= 0 {
		return defaultMaxFileBytes
	}
	return r.MaxFileBytes
}

func (r *Reader) recordIssue(issue ReadIssue) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issues[issue.Path] = issue
}

func cleanLogicalPath(path string) string {
	return filepath.Clean(string(filepath.Separator) + path)
}

func readFileBounded(abs string, logical string, limit int64) ([]byte, error) {
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lr := &io.LimitedReader{R: f, N: limit + 1}
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, &ErrFileTooLarge{Path: logical, Limit: limit}
	}
	return b, nil
}
