// Package archiver provides tar capabilies.
package archiver
/*
#cgo LDFLAGS: -lpacificauploaderserver

#include <archiver.h>
#include <malloc.h>
*/
import "C"
import (
	"unsafe"
)

func archive(tar int, sourcefile int, src string, size int64, mtime int64) int {
    srcName := C.CString(src)
    defer C.free(unsafe.Pointer(srcName))

    error := C.pacifica_uploader_server_archiver(C.int(tar), C.int(sourcefile), srcName, C.long(size), C.long(mtime))
    return int(error)
}

