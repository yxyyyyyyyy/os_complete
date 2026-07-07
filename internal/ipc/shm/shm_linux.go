//go:build linux

package shm

import (
	"bytes"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	memfdCloexec   = 0x0001
	sysMemfdCreate = 319
)

func exerciseMemoryTransport(payload []byte, workerCount int) (SmokeResult, error) {
	if workerCount <= 0 {
		workerCount = 1
	}
	result := SmokeResult{
		IPCMode:                "memfd-mmap",
		EvidenceMode:           "real-shm-ipc",
		PayloadBytesSent:       len(payload),
		ReferencedContextBytes: len(payload) * workerCount,
		SharedPages:            1,
	}
	fd, err := memfdCreate("aort_context_page")
	if err != nil {
		return result, fmt.Errorf("memfd_create failed: %w", err)
	}
	result.MemfdCreateSuccess = true
	cleanupOK := true
	defer func() {
		if err := syscall.Close(fd); err != nil {
			cleanupOK = false
		}
	}()
	if err := syscall.Ftruncate(fd, int64(len(payload))); err != nil {
		return result, fmt.Errorf("ftruncate failed: %w", err)
	}
	mapped, err := syscall.Mmap(fd, 0, len(payload), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return result, fmt.Errorf("mmap writer failed: %w", err)
	}
	result.MmapSuccess = true
	copy(mapped, payload)
	if err := syscall.Munmap(mapped); err != nil {
		cleanupOK = false
	}

	for i := 0; i < workerCount; i++ {
		ok, err := workerReadViaFDPassing(fd, payload)
		if err != nil {
			return result, err
		}
		result.FDPassingSuccess = true
		result.WorkerMmapSuccess = true
		if !ok {
			result.DataIntegrityOK = false
			return result, nil
		}
	}
	result.DataIntegrityOK = true
	result.AvoidedCopyBytes = len(payload) * max(0, workerCount-1)
	result.CleanupSuccess = cleanupOK
	return result, nil
}

func workerReadViaFDPassing(fd int, want []byte) (bool, error) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return false, fmt.Errorf("socketpair failed: %w", err)
	}
	defer syscall.Close(pair[0])
	defer syscall.Close(pair[1])
	rights := syscall.UnixRights(fd)
	if err := syscall.Sendmsg(pair[0], []byte{1}, rights, nil, 0); err != nil {
		return false, fmt.Errorf("sendmsg fd failed: %w", err)
	}
	buf := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4))
	_, oobn, _, _, err := syscall.Recvmsg(pair[1], buf, oob, 0)
	if err != nil {
		return false, fmt.Errorf("recvmsg fd failed: %w", err)
	}
	messages, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return false, fmt.Errorf("parse control message failed: %w", err)
	}
	if len(messages) == 0 {
		return false, fmt.Errorf("no fd received")
	}
	fds, err := syscall.ParseUnixRights(&messages[0])
	if err != nil {
		return false, fmt.Errorf("parse unix rights failed: %w", err)
	}
	if len(fds) == 0 {
		return false, fmt.Errorf("empty unix rights")
	}
	workerFD := fds[0]
	defer syscall.Close(workerFD)
	mapped, err := syscall.Mmap(workerFD, 0, len(want), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return false, fmt.Errorf("worker mmap failed: %w", err)
	}
	defer syscall.Munmap(mapped)
	return bytes.Equal(mapped, want), nil
}

func memfdCreate(name string) (int, error) {
	raw := append([]byte(name), 0)
	fd, _, errno := syscall.Syscall(sysMemfdCreate, uintptr(unsafe.Pointer(&raw[0])), memfdCloexec, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
