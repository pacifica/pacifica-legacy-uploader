// Package archiver provides tar capabilies.
package archiver
/*
#cgo LDFLAGS: -lpacificauploaderserver

#include <windows.h>
#include <io.h>
#include <fcntl.h>
#include <archiver.h>

int archive_w(HANDLE tarFD, HANDLE sourcefile, const char * srcName, long size, long mtime )
{
   int             fd = _open_osfhandle((intptr_t)tarFD, _O_WRONLY);
   int             fd2 = _open_osfhandle((intptr_t)sourcefile, _O_RDONLY);
   if(fd == -1 || fd2 == -1)
   {
	return -1;
   }
   return pacifica_uploader_server_archiver(fd, fd2, srcName, size, mtime);
}
*/
import "C"

import (
	"os"
	"unsafe"
)

func Archive(tar *os.File, sourcefile *os.File, src string, size int64, mtime int64) int {
    srcName := C.CString(src)
    defer C.free(unsafe.Pointer(srcName))

    error := C.archive_w((C.HANDLE)(tar.Fd()), (C.HANDLE)(sourcefile.Fd()), srcName, C.long(size), C.long(mtime))
    return int(error)
}

