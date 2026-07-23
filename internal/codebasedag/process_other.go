//go:build !linux

package codebasedag

func NewDefaultProcessRuntime() ProcessRuntime {
	return unsupportedProcessRuntime{reason: "real cross-process worker lifecycle requires linux"}
}
