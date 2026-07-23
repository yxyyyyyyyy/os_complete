package codebasedag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const artifactHashIndex = "ARTIFACT_SHA256.json"

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type RunStore struct {
	Root  string
	RunID string
	Dir   string
	mu    sync.Mutex
}

func NewRunStore(root, runID string) (*RunStore, error) {
	if !runIDPattern.MatchString(runID) || runID == "." || runID == ".." {
		return nil, fmt.Errorf("invalid run ID %q", runID)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	dir := filepath.Join(root, runID)
	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("run directory already exists: %s", dir)
		}
		return nil, err
	}
	return &RunStore{Root: root, RunID: runID, Dir: dir}, nil
}

func (s *RunStore) WriteJSON(rel string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return s.WriteBytes(rel, data, 0o600)
}

func (s *RunStore) WriteBytes(rel string, data []byte, mode os.FileMode) error {
	path, err := s.safePath(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := s.rejectSymlinkAncestors(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func (s *RunStore) AppendJSONL(rel string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.safePath(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := s.rejectSymlinkAncestors(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func (s *RunStore) FinalizeHashes() (map[string]string, error) {
	hashes := make(map[string]string)
	err := filepath.WalkDir(s.Dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == s.Dir {
			return nil
		}
		rel, err := filepath.Rel(s.Dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == artifactHashIndex {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact path %q is a symlink", rel)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("artifact path %q is not a regular file", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		hashes[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return nil, err
	}
	ordered := make(map[string]string, len(hashes))
	keys := make([]string, 0, len(hashes))
	for key := range hashes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		ordered[key] = hashes[key]
	}
	if err := s.WriteJSON(artifactHashIndex, ordered); err != nil {
		return nil, err
	}
	return ordered, nil
}

func (s *RunStore) safePath(rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, `\`) {
		return "", fmt.Errorf("invalid artifact path %q", rel)
	}
	clean := filepath.Clean(rel)
	if clean != rel || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid artifact path %q", rel)
	}
	path := filepath.Join(s.Dir, clean)
	back, err := filepath.Rel(s.Dir, path)
	if err != nil {
		return "", err
	}
	if back == ".." || strings.HasPrefix(back, "../") {
		return "", fmt.Errorf("artifact path escapes run directory: %q", rel)
	}
	return path, nil
}

func (s *RunStore) rejectSymlinkAncestors(path string) error {
	dir := filepath.Dir(path)
	for {
		if dir == s.Dir || dir == "." || dir == string(filepath.Separator) {
			break
		}
		info, err := os.Lstat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact parent %q is a symlink", dir)
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("artifact path %q is a symlink", path)
	}
	return nil
}
