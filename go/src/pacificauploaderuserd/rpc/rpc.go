package rpc

import (
	"time"
)

const (
	ACCESS string = "Server.Access"
	BUNDLEFILE string = "Server.BundleFile"
)

type BundleFileError int

const (
	BundleFileError_OK BundleFileError = iota
	BundleFileError_FNF
	BundleFileError_FBAD
	BundleFileError_BNF
	BundleFileError_PERM
	BundleFileError_BPERM
	BundleFileError_UNKNOWN
	BundleFileError_CORRUPT
	BundleFileError_CHANGED
)

type AccessArgs struct {
	Path string
}

type BundleFileArgs struct {
	BundlePath string
	LocalPath string
	Name string
}

type BundleFileResult struct {
	Retval bool
	Sha1 string
	Error BundleFileError
	Mtime time.Time
}

//Is it an error?
func BundleFileErrorIsError(fe BundleFileError) bool {
	if fe != BundleFileError_OK {
		return true
	}
	return false
}

//Errors that can be corrected with another main loop through the bundler.
func BundleFileErrorIsTransient(fe BundleFileError, transient_permanent bool) bool {
	if fe == BundleFileError_UNKNOWN || (transient_permanent && fe == BundleFileError_CHANGED) {
		return true
	}
	return false
}

//Errors specific to a file that could get a single file kicked out of the bundle.
func BundleFileErrorIsTransientError(fe BundleFileError) bool {
	if fe == BundleFileError_PERM || fe == BundleFileError_FBAD || fe== BundleFileError_FNF {
		return true
	}
	return false
}

//Errors that would cause a bundle to be set to Error state.
func BundleFileErrorIsPermanentError(fe BundleFileError, transient_permanent bool) bool {
	if (transient_permanent && BundleFileErrorIsTransientError(fe)) || fe == BundleFileError_BNF || fe == BundleFileError_BPERM || fe == BundleFileError_CORRUPT {
		return true
	}
	return false
}
