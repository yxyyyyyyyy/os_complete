//go:build !linux

package shm

import "fmt"

func exerciseMemoryTransport(_ []byte, _ int) (SmokeResult, error) {
	return SmokeResult{}, fmt.Errorf("not linux: memfd/mmap shared-memory IPC requires Linux")
}
