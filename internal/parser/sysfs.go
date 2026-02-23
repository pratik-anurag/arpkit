package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Reader is a small cached filesystem reader used for sysfs/procfs parsing.
type Reader struct {
	Root string

	mu    sync.Mutex
	cache map[string][]byte
}

func NewReader(root string) *Reader {
	if root == "" {
		root = "/"
	}
	return &Reader{
		Root:  root,
		cache: make(map[string][]byte, 512),
	}
}

func (r *Reader) ReadFile(path string) (string, error) {
	abs := r.resolve(path)

	r.mu.Lock()
	if b, ok := r.cache[abs]; ok {
		r.mu.Unlock()
		return strings.TrimSpace(string(b)), nil
	}
	r.mu.Unlock()

	b, err := os.ReadFile(abs)
	if err != nil {
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

func (r *Reader) resolve(path string) string {
	clean := filepath.Clean(path)
	clean = strings.TrimPrefix(clean, string(filepath.Separator))
	if r.Root == "/" {
		return string(filepath.Separator) + clean
	}
	return filepath.Join(r.Root, clean)
}
