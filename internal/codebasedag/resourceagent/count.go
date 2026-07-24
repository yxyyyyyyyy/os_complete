package resourceagent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func CountGeneratedFiles() int {
	_, file, _, ok := runtime.Caller(0)
	if !ok { return 0 }
	root := filepath.Dir(file)
	n := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil { return nil }
		if d.IsDir() {
			if d.Name() == "_broken" { return filepath.SkipDir }
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "gen_") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			n++
		}
		return nil
	})
	return n
}

func PhysicalLinesApprox() int {
	_, file, _, ok := runtime.Caller(0)
	if !ok { return 0 }
	root := filepath.Dir(file)
	total := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil { return nil }
		if d.IsDir() {
			if d.Name() == "_broken" { return filepath.SkipDir }
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") { return nil }
		b, err := os.ReadFile(path)
		if err != nil { return nil }
		total += strings.Count(string(b), "\n")
		if len(b) > 0 && !strings.HasSuffix(string(b), "\n") { total++ }
		return nil
	})
	return total
}
