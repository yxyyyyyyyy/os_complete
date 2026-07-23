package codebasedag

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DefaultMinPhysicalLines = 30000
	DefaultMinNonblankLines = 30000
)

type ManifestOptions struct {
	MinPhysical int
	MinNonblank int
	GitPath     string
}

func BuildSourceManifest(ctx context.Context, dir string) (SourceManifest, []SeedFile, error) {
	return BuildSourceManifestWithOptions(ctx, dir, ManifestOptions{})
}

func BuildSourceManifestWithOptions(ctx context.Context, dir string, opts ManifestOptions) (SourceManifest, []SeedFile, error) {
	gitPath := opts.GitPath
	if gitPath == "" {
		gitPath = "git"
	}
	commit, err := gitOutput(ctx, gitPath, dir, "rev-parse", "HEAD")
	if err != nil {
		return SourceManifest{}, nil, err
	}
	status, err := gitOutput(ctx, gitPath, dir, "status", "--porcelain")
	if err != nil {
		return SourceManifest{}, nil, err
	}
	list, err := gitOutputBytes(ctx, gitPath, dir, "ls-files", "-z")
	if err != nil {
		return SourceManifest{}, nil, err
	}

	manifest := SourceManifest{
		SchemaVersion: SchemaVersion,
		GitCommit:     strings.TrimSpace(commit),
		GitDirty:      strings.TrimSpace(status) != "",
	}
	seen := make(map[string]struct{})
	var seeds []SeedFile
	for _, raw := range bytes.Split(list, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		rel := string(raw)
		clean, err := cleanManifestPath(rel)
		if err != nil {
			return SourceManifest{}, nil, err
		}
		if _, ok := seen[clean]; ok {
			return SourceManifest{}, nil, fmt.Errorf("duplicate tracked path %q", clean)
		}
		seen[clean] = struct{}{}

		abs := filepath.Join(dir, filepath.FromSlash(clean))
		info, err := os.Lstat(abs)
		if err != nil {
			return SourceManifest{}, nil, fmt.Errorf("stat tracked path %q: %w", clean, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return SourceManifest{}, nil, fmt.Errorf("tracked path %q is a symlink", clean)
		}
		if !info.Mode().IsRegular() {
			return SourceManifest{}, nil, fmt.Errorf("tracked path %q is not a regular file", clean)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return SourceManifest{}, nil, fmt.Errorf("read tracked path %q: %w", clean, err)
		}
		seeds = append(seeds, SeedFile{Path: clean, Mode: info.Mode(), Data: bytes.Clone(data)})
		if filepath.Ext(clean) != ".go" {
			continue
		}
		sum := sha256.Sum256(data)
		physical, nonblank, err := countGoLines(data)
		if err != nil {
			return SourceManifest{}, nil, fmt.Errorf("count %q: %w", clean, err)
		}
		manifest.Files = append(manifest.Files, SourceFile{
			Path:          clean,
			SHA256:        hex.EncodeToString(sum[:]),
			Bytes:         int64(len(data)),
			PhysicalLines: physical,
			NonblankLines: nonblank,
		})
		manifest.PhysicalLines += physical
		manifest.NonblankLines += nonblank
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	sort.Slice(seeds, func(i, j int) bool { return seeds[i].Path < seeds[j].Path })
	manifest.TrackedGoFiles = len(manifest.Files)
	manifest.TreeHash = sourceTreeHash(manifest.Files)
	if err := manifest.ValidateLargeCodebaseWithOptions(opts); err != nil {
		return SourceManifest{}, nil, err
	}
	return manifest, seeds, nil
}

func (m SourceManifest) ValidateLargeCodebase() error {
	return m.ValidateLargeCodebaseWithOptions(ManifestOptions{})
}

func (m SourceManifest) ValidateLargeCodebaseWithOptions(opts ManifestOptions) error {
	minPhysical, minNonblank := normalizedManifestThresholds(opts)
	if m.PhysicalLines < minPhysical {
		return fmt.Errorf("tracked Go physical lines %d below minimum %d", m.PhysicalLines, minPhysical)
	}
	if m.NonblankLines < minNonblank {
		return fmt.Errorf("tracked Go nonblank lines %d below minimum %d", m.NonblankLines, minNonblank)
	}
	return nil
}

func normalizedManifestThresholds(opts ManifestOptions) (int, int) {
	minPhysical := opts.MinPhysical
	if minPhysical == 0 {
		minPhysical = DefaultMinPhysicalLines
	}
	minNonblank := opts.MinNonblank
	if minNonblank == 0 {
		minNonblank = DefaultMinNonblankLines
	}
	return minPhysical, minNonblank
}

func gitOutput(ctx context.Context, gitPath, dir string, args ...string) (string, error) {
	out, err := gitOutputBytes(ctx, gitPath, dir, args...)
	return string(out), err
}

func gitOutputBytes(ctx context.Context, gitPath, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, gitPath, append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w", args, err)
	}
	return out, nil
}

func cleanManifestPath(path string) (string, error) {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, `\`) {
		return "", fmt.Errorf("invalid tracked path %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid tracked path %q", path)
	}
	return clean, nil
}

func countGoLines(data []byte) (int, int, error) {
	if len(data) == 0 {
		return 0, 0, nil
	}
	physical := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		physical++
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	nonblank := 0
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			nonblank++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	return physical, nonblank, nil
}

func sourceTreeHash(files []SourceFile) string {
	hash := sha256.New()
	for _, file := range files {
		hash.Write([]byte(file.Path))
		hash.Write([]byte{0})
		hash.Write([]byte(file.SHA256))
		hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
