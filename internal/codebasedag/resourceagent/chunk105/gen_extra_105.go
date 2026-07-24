package chunk105

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Constants for supported checksum algorithms.
const (
	AlgoSHA256_105 = "sha256"
	AlgoMD5_105    = "md5"
)

// ChecksumEntry_105 holds the computed checksum for a single file.
type ChecksumEntry_105 struct {
	Path      string // relative path from workspace root
	Algorithm string // hash algorithm used
	Value     []byte // raw checksum bytes
	Size      int64  // file size in bytes
}

// ChecksumTable_105 maps relative file paths to their checksum entries.
type ChecksumTable_105 map[string]ChecksumEntry_105

// NewChecksumEntry_105 creates a new ChecksumEntry with validated fields.
func NewChecksumEntry_105(path, algorithm string, value []byte, size int64) (ChecksumEntry_105, error) {
	if !IsValidAlgorithm_105(algorithm) {
		return ChecksumEntry_105{}, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
	if len(value) == 0 {
		return ChecksumEntry_105{}, errors.New("checksum value must not be empty")
	}
	if size < 0 {
		return ChecksumEntry_105{}, errors.New("file size must not be negative")
	}
	return ChecksumEntry_105{
		Path:      path,
		Algorithm: algorithm,
		Value:     value,
		Size:      size,
	}, nil
}

// GetSupportedAlgorithms_105 returns the list of checksum algorithms supported.
func GetSupportedAlgorithms_105() []string {
	return []string{AlgoSHA256_105, AlgoMD5_105}
}

// IsValidAlgorithm_105 checks if the given algorithm name is supported.
func IsValidAlgorithm_105(algo string) bool {
	switch algo {
	case AlgoSHA256_105, AlgoMD5_105:
		return true
	}
	return false
}

// newHash_105 returns a new hash.Hash for the given algorithm.
func newHash_105(algo string) (hash.Hash, error) {
	switch algo {
	case AlgoSHA256_105:
		return sha256.New(), nil
	case AlgoMD5_105:
		return md5.New(), nil
	}
	return nil, fmt.Errorf("unknown algorithm: %s", algo)
}

// hashSize_105 returns the expected checksum byte length for the algorithm.
func hashSize_105(algo string) int {
	switch algo {
	case AlgoSHA256_105:
		return sha256.Size
	case AlgoMD5_105:
		return md5.Size
	}
	return 0
}

// ComputeFileChecksum_105 reads a file and computes its checksum using the given algorithm.
func ComputeFileChecksum_105(filePath, algorithm string) (ChecksumEntry_105, error) {
	if !IsValidAlgorithm_105(algorithm) {
		return ChecksumEntry_105{}, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
	f, err := os.Open(filePath)
	if err != nil {
		return ChecksumEntry_105{}, fmt.Errorf("cannot open file %s: %w", filePath, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return ChecksumEntry_105{}, fmt.Errorf("cannot stat file %s: %w", filePath, err)
	}
	if fi.IsDir() {
		return ChecksumEntry_105{}, fmt.Errorf("%s is a directory, not a file", filePath)
	}

	h, err := newHash_105(algorithm)
	if err != nil {
		return ChecksumEntry_105{}, err
	}
	if _, err := io.Copy(h, f); err != nil {
		return ChecksumEntry_105{}, fmt.Errorf("error reading file %s: %w", filePath, err)
	}
	value := h.Sum(nil)

	// Compute the relative path from the workspace root.
	// If no workspace root is provided, use the file's base name as relative.
	// We use the file name as path if no context is given.
	relPath := filePath
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, filePath); err == nil {
			relPath = rel
		}
	}

	return NewChecksumEntry_105(relPath, algorithm, value, fi.Size())
}

// ComputeDirectoryChecksums_105 walks through a directory and computes checksums for all regular files.
func ComputeDirectoryChecksums_105(dirPath, algorithm string) (ChecksumTable_105, error) {
	table := make(ChecksumTable_105)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		entry, err := ComputeFileChecksum_105(path, algorithm)
		if err != nil {
			return fmt.Errorf("failed checksum for %s: %w", path, err)
		}
		table[entry.Path] = entry
		return nil
	})
	if err != nil {
		return nil, err
	}
	return table, nil
}

// ReadChecksumFile_105 reads a checksum file and returns a ChecksumTable.
// Expected format: each non-empty line: <algorithm>:<hex>:<size>:<relative_path>
func ReadChecksumFile_105(filePath string) (ChecksumTable_105, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open checksum file %s: %w", filePath, err)
	}
	defer f.Close()

	table := make(ChecksumTable_105)
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNo++
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("line %d: invalid format, expected 4 colon-separated fields", lineNo)
		}
		algo := parts[0]
		if !IsValidAlgorithm_105(algo) {
			return nil, fmt.Errorf("line %d: unsupported algorithm %s", lineNo, algo)
		}
		hexVal := parts[1]
		value, err := hex.DecodeString(hexVal)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid hex checksum: %s", lineNo, hexVal)
		}
		var size int64
		if _, err := fmt.Sscanf(parts[2], "%d", &size); err != nil {
			return nil, fmt.Errorf("line %d: invalid file size: %s", lineNo, parts[2])
		}
		relPath := parts[3]
		// Normalize path separators to forward slash for consistency.
		relPath = filepath.ToSlash(relPath)
		entry, err := NewChecksumEntry_105(relPath, algo, value, size)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid entry: %w", lineNo, err)
		}
		table[relPath] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading checksum file: %w", err)
	}
	return table, nil
}

// WriteChecksumFile_105 writes a checksum table to a file in colon-separated format.
func WriteChecksumFile_105(table ChecksumTable_105, filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("cannot create checksum file %s: %w", filePath, err)
	}
	defer f.Close()

	// Sort paths for deterministic output.
	paths := make([]string, 0, len(table))
	for p := range table {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		entry := table[p]
		hexVal := hex.EncodeToString(entry.Value)
		line := fmt.Sprintf("%s:%s:%d:%s\n", entry.Algorithm, hexVal, entry.Size, entry.Path)
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("error writing to checksum file: %w", err)
		}
	}
	return nil
}

// ValidateChecksumTable_105 verifies that every entry in the table matches the actual file content.
// It returns a list of paths that have mismatched checksums.
func ValidateChecksumTable_105(table ChecksumTable_105, baseDir string) ([]string, error) {
	var mismatches []string
	for relPath, expected := range table {
		fullPath := filepath.Join(baseDir, relPath)
		entry, err := ComputeFileChecksum_105(fullPath, expected.Algorithm)
		if err != nil {
			// If file not found, it's a mismatch
			if os.IsNotExist(err) {
				mismatches = append(mismatches, relPath)
				continue
			}
			return nil, fmt.Errorf("error computing checksum for %s: %w", fullPath, err)
		}
		if !checksumsEqual_105(entry.Value, expected.Value) {
			mismatches = append(mismatches, relPath)
		}
	}
	return mismatches, nil
}

// checksumsEqual_105 compares two checksum byte slices safely.
func checksumsEqual_105(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// VerifySingleFile_105 checks if the given file matches the expected checksum entry.
func VerifySingleFile_105(filePath string, expected ChecksumEntry_105) (bool, error) {
	entry, err := ComputeFileChecksum_105(filePath, expected.Algorithm)
	if err != nil {
		return false, fmt.Errorf("cannot compute checksum for %s: %w", filePath, err)
	}
	return checksumsEqual_105(entry.Value, expected.Value) && entry.Size == expected.Size, nil
}

// VerifyDirectory_105 validates all files in a directory against a checksum table.
// Returns list of files that fail verification.
func VerifyDirectory_105(dirPath string, expectedTable ChecksumTable_105) ([]string, error) {
	var fails []string
	for relPath, expected := range expectedTable {
		fullPath := filepath.Join(dirPath, relPath)
		ok, err := VerifySingleFile_105(fullPath, expected)
		if err != nil {
			if os.IsNotExist(err) {
				fails = append(fails, relPath)
				continue
			}
			return nil, fmt.Errorf("verification error for %s: %w", relPath, err)
		}
		if !ok {
			fails = append(fails, relPath)
		}
	}
	return fails, nil
}

// MergeChecksumTables_105 merges multiple checksum tables into one.
// Later tables override earlier ones for the same key.
func MergeChecksumTables_105(tables ...ChecksumTable_105) ChecksumTable_105 {
	result := make(ChecksumTable_105)
	for _, t := range tables {
		for k, v := range t {
			result[k] = v
		}
	}
	return result
}

// FilterChecksumTable_105 returns a new table containing only entries whose path starts with the given prefix.
func FilterChecksumTable_105(table ChecksumTable_105, prefix string) ChecksumTable_105 {
	prefix = filepath.ToSlash(prefix)
	filtered := make(ChecksumTable_105)
	for path, entry := range table {
		if strings.HasPrefix(path, prefix) {
			filtered[path] = entry
		}
	}
	return filtered
}

// ChecksumTableToString_105 serializes a checksum table to a string (one entry per line).
func ChecksumTableToString_105(table ChecksumTable_105) string {
	paths := make([]string, 0, len(table))
	for p := range table {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		entry := table[p]
		hexVal := hex.EncodeToString(entry.Value)
		fmt.Fprintf(&b, "%s:%s:%d:%s\n", entry.Algorithm, hexVal, entry.Size, entry.Path)
	}
	return b.String()
}

// StringToChecksumTable_105 deserializes a string in the same format as ChecksumTableToString_105.
func StringToChecksumTable_105(s string) (ChecksumTable_105, error) {
	table := make(ChecksumTable_105)
	scanner := bufio.NewScanner(strings.NewReader(s))
	lineNo := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNo++
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("line %d: invalid format", lineNo)
		}
		algo := parts[0]
		if !IsValidAlgorithm_105(algo) {
			return nil, fmt.Errorf("line %d: unsupported algorithm %s", lineNo, algo)
		}
		hexVal := parts[1]
		value, err := hex.DecodeString(hexVal)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid hex: %s", lineNo, hexVal)
		}
		var size int64
		if _, err := fmt.Sscanf(parts[2], "%d", &size); err != nil {
			return nil, fmt.Errorf("line %d: invalid size: %s", lineNo, parts[2])
		}
		relPath := parts[3]
		relPath = filepath.ToSlash(relPath)
		entry, err := NewChecksumEntry_105(relPath, algo, value, size)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid entry: %w", lineNo, err)
		}
		table[relPath] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing string: %w", err)
	}
	return table, nil
}

// GenerateChecksumTableFromGlob_105 computes checksums for all files matching the glob pattern.
func GenerateChecksumTableFromGlob_105(pattern, algorithm string) (ChecksumTable_105, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob error: %w", err)
	}
	table := make(ChecksumTable_105)
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %s: %w", path, err)
		}
		if info.IsDir() {
			continue
		}
		entry, err := ComputeFileChecksum_105(path, algorithm)
		if err != nil {
			return nil, fmt.Errorf("cannot compute checksum for %s: %w", path, err)
		}
		table[entry.Path] = entry
	}
	return table, nil
}

// CompareChecksumTables_105 compares two checksum tables and returns lists of added, removed, and modified paths.
// Paths present in t2 but not in t1 are added; in t1 but not t2 are removed; in both but with different checksums are modified.
func CompareChecksumTables_105(t1, t2 ChecksumTable_105) (added, removed, modified []string, err error) {
	// Create sets of paths
	paths1 := make(map[string]bool)
	for p := range t1 {
		paths1[p] = true
	}
	paths2 := make(map[string]bool)
	for p := range t2 {
		paths2[p] = true
	}

	for p := range paths2 {
		if !paths1[p] {
			added = append(added, p)
		} else {
			if !checksumsEqual_105(t1[p].Value, t2[p].Value) || t1[p].Size != t2[p].Size {
				modified = append(modified, p)
			}
		}
	}
	for p := range paths1 {
		if !paths2[p] {
			removed = append(removed, p)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(modified)
	return
}

// ValidateLowerdirIntegrity_105 is a high-level function that validates the integrity of a lowerdir
// against a checksum file. It reads the checksum file, computes checksums for all files in the directory,
// and reports mismatches.
func ValidateLowerdirIntegrity_105(lowerdir, checksumFilePath string) error {
	expectedTable, err := ReadChecksumFile_105(checksumFilePath)
	if err != nil {
		return fmt.Errorf("cannot read expected checksums: %w", err)
	}
	actualTable, err := ComputeDirectoryChecksums_105(lowerdir, AlgoSHA256_105)
	if err != nil {
		return fmt.Errorf("cannot compute actual checksums: %w", err)
	}
	added, removed, modified, err := CompareChecksumTables_105(expectedTable, actualTable)
	if err != nil {
		return err
	}
	if len(added) > 0 || len(removed) > 0 || len(modified) > 0 {
		return fmt.Errorf("integrity check failed: added %d, removed %d, modified %d", len(added), len(removed), len(modified))
	}
	return nil
}

// VerifyLowerdirChecksums_105 verifies each file in lowerdir against the provided checksum file.
// It returns a detailed error with mismatches if any.
func VerifyLowerdirChecksums_105(lowerdir, checksumFile string) error {
	table, err := ReadChecksumFile_105(checksumFile)
	if err != nil {
		return fmt.Errorf("cannot read checksum file: %w", err)
	}
	mismatches, err := ValidateChecksumTable_105(table, lowerdir)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("checksum mismatch for %d files: %v", len(mismatches), mismatches)
	}
	return nil
}

// UpdateChecksumFile_105 recomputes checksums for all files in the given directory and writes a new checksum file.
func UpdateChecksumFile_105(dirPath, checksumFile, algorithm string) error {
	table, err := ComputeDirectoryChecksums_105(dirPath, algorithm)
	if err != nil {
		return fmt.Errorf("cannot compute checksums: %w", err)
	}
	if err := WriteChecksumFile_105(table, checksumFile); err != nil {
		return fmt.Errorf("cannot write checksum file: %w", err)
	}
	return nil
}

// PrintChecksumTable_105 prints the checksum table to stdout (for debugging).
func PrintChecksumTable_105(table ChecksumTable_105) {
	fmt.Print(ChecksumTableToString_105(table))
}

// CopyChecksumTable_105 creates a deep copy of a checksum table.
func CopyChecksumTable_105(src ChecksumTable_105) ChecksumTable_105 {
	dst := make(ChecksumTable_105, len(src))
	for k, v := range src {
		valCopy := make([]byte, len(v.Value))
		copy(valCopy, v.Value)
		dst[k] = ChecksumEntry_105{
			Path:      v.Path,
			Algorithm: v.Algorithm,
			Value:     valCopy,
			Size:      v.Size,
		}
	}
	return dst
}

// DiffChecksumTables_105 returns a human-readable diff between two checksum tables.
func DiffChecksumTables_105(t1, t2 ChecksumTable_105) string {
	added, removed, modified, _ := CompareChecksumTables_105(t1, t2)
	var b strings.Builder
	if len(added) > 0 {
		fmt.Fprintf(&b, "Added files:\n")
		for _, p := range added {
			fmt.Fprintf(&b, "  + %s\n", p)
		}
	}
	if len(removed) > 0 {
		fmt.Fprintf(&b, "Removed files:\n")
		for _, p := range removed {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}
	if len(modified) > 0 {
		fmt.Fprintf(&b, "Modified files:\n")
		for _, p := range modified {
			fmt.Fprintf(&b, "  ~ %s (old: %s new: %s)\n", p, hex.EncodeToString(t1[p].Value), hex.EncodeToString(t2[p].Value))
		}
	}
	if b.Len() == 0 {
		b.WriteString("No differences.\n")
	}
	return b.String()
}

// FilesizeFromTable_105 returns the total size of all files in the table.
func FilesizeFromTable_105(table ChecksumTable_105) int64 {
	var total int64
	for _, entry := range table {
		total += entry.Size
	}
	return total
}

// CountFilesInTable_105 returns the number of entries in the table.
func CountFilesInTable_105(table ChecksumTable_105) int {
	return len(table)
}

// ChecksumEntryToString_105 returns a string representation of a single entry.
func ChecksumEntryToString_105(entry ChecksumEntry_105) string {
	return fmt.Sprintf("%s:%s:%d:%s", entry.Algorithm, hex.EncodeToString(entry.Value), entry.Size, entry.Path)
}

// ParseChecksumEntryFromString_105 parses a single line into a ChecksumEntry.
func ParseChecksumEntryFromString_105(line string) (ChecksumEntry_105, error) {
	parts := strings.SplitN(line, ":", 4)
	if len(parts) != 4 {
		return ChecksumEntry_105{}, errors.New("invalid entry format")
	}
	algo := parts[0]
	if !IsValidAlgorithm_105(algo) {
		return ChecksumEntry_105{}, fmt.Errorf("unsupported algorithm: %s", algo)
	}
	value, err := hex.DecodeString(parts[1])
	if err != nil {
		return ChecksumEntry_105{}, fmt.Errorf("invalid hex: %s", parts[1])
	}
	var size int64
	if _, err := fmt.Sscanf(parts[2], "%d", &size); err != nil {
		return ChecksumEntry_105{}, fmt.Errorf("invalid size: %s", parts[2])
	}
	relPath := filepath.ToSlash(parts[3])
	return NewChecksumEntry_105(relPath, algo, value, size)
}

// ChecksumTableToMapByAlgorithm_105 groups entries by algorithm.
func ChecksumTableToMapByAlgorithm_105(table ChecksumTable_105) map[string]ChecksumTable_105 {
	byAlgo := make(map[string]ChecksumTable_105)
	for path, entry := range table {
		if _, ok := byAlgo[entry.Algorithm]; !ok {
			byAlgo[entry.Algorithm] = make(ChecksumTable_105)
		}
		byAlgo[entry.Algorithm][path] = entry
	}
	return byAlgo
}

// HasChecksumEntry_105 checks if a given path exists in the table.
func HasChecksumEntry_105(table ChecksumTable_105, path string) bool {
	_, ok := table[filepath.ToSlash(path)]
	return ok
}

// GetChecksumEntry_105 returns the entry for a path, or an error if not found.
func GetChecksumEntry_105(table ChecksumTable_105, path string) (ChecksumEntry_105, error) {
	entry, ok := table[filepath.ToSlash(path)]
	if !ok {
		return ChecksumEntry_105{}, fmt.Errorf("path %s not found in checksum table", path)
	}
	return entry, nil
}

// RemoveChecksumEntry_105 deletes an entry from the table and returns the deleted entry.
func RemoveChecksumEntry_105(table ChecksumTable_105, path string) (ChecksumEntry_105, bool) {
	entry, ok := table[filepath.ToSlash(path)]
	if ok {
		delete(table, filepath.ToSlash(path))
	}
	return entry, ok
}

// AddChecksumEntry_105 adds or updates an entry in the table.
func AddChecksumEntry_105(table ChecksumTable_105, entry ChecksumEntry_105) {
	table[entry.Path] = entry
}

// KeysOfChecksumTable_105 returns all keys (paths) sorted.
func KeysOfChecksumTable_105(table ChecksumTable_105) []string {
	keys := make([]string, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ValuesOfChecksumTable_105 returns all entries (unsorted).
func ValuesOfChecksumTable_105(table ChecksumTable_105) []ChecksumEntry_105 {
	values := make([]ChecksumEntry_105, 0, len(table))
	for _, v := range table {
		values = append(values, v)
	}
	return values
}
