//go:build windows || plan9

package utils

func EnsureFileDescriptorLimit(_ uint64) {
	// no-op on unsupported platforms
}
