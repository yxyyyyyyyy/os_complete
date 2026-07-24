package codebasedag

import "fmt"

func errJudge(msg string) error {
	return fmt.Errorf("judge task: %s", msg)
}
