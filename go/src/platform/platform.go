package platform

import (
	"runtime"
)

type PlatformCode int

const (
	Unknown PlatformCode = iota
	Windows
	Linux
	Darwin
	Freebsd
	Openbsd
)

// GetPlatform returns one of the OS constants defined in
// this package.  It uses runtime.GOOS to determine the OS
func PlatformGet() PlatformCode {
	goos := runtime.GOOS
	switch {
	case goos == "windows":
		return Windows
	case goos == "linux":
		return Linux
	case goos == "darwin":
		return Darwin
	case goos == "freebsd":
		return Freebsd
	case goos == "openbsd":
		return Openbsd
	}
	return Unknown
}
