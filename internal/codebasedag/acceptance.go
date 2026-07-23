package codebasedag

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed acceptance/*.sh
var acceptanceFS embed.FS

type AcceptanceScript struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

func MaterializeAcceptance(dir string) ([]AcceptanceScript, error) {
	entries, err := fs.ReadDir(acceptanceFS, "acceptance")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var records []AcceptanceScript
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip macOS AppleDouble junk that can appear after tar/scp from Darwin.
		if strings.HasPrefix(name, "._") {
			continue
		}
		embeddedPath := filepath.ToSlash(filepath.Join("acceptance", name))
		data, err := acceptanceFS.ReadFile(embeddedPath)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(dir, name)
		if _, err := os.Lstat(target); err == nil {
			return nil, fmt.Errorf("acceptance script already exists: %s", target)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if err := os.WriteFile(target, data, 0o500); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		records = append(records, AcceptanceScript{
			Path:   embeddedPath,
			SHA256: hex.EncodeToString(sum[:]),
			Bytes:  len(data),
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	return records, nil
}

func VerifyAcceptance(dir string, records []AcceptanceScript) error {
	for _, record := range records {
		path := filepath.Join(dir, filepath.Base(record.Path))
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("acceptance script %q is a symlink", path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("acceptance script %q is not regular", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != record.SHA256 {
			return fmt.Errorf("acceptance script %q hash %s, want %s", path, got, record.SHA256)
		}
	}
	return nil
}
