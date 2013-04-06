// Package archiver provides tar capabilies.
package archiver
/*
#cgo windows CFLAGS: -fno-stack-check -fno-stack-protector -mno-stack-arg-probe

#include <windows.h>
#include <io.h>
#include <fcntl.h>

int archive(int tarFD, int sourcefile, const char * srcName, long size, long mtime );
int archive_w(HANDLE tarFD, HANDLE sourcefile, const char * srcName, long size, long mtime )
{
   int             fd = _open_osfhandle((intptr_t)tarFD, _O_WRONLY);
   int             fd2 = _open_osfhandle((intptr_t)sourcefile, _O_RDONLY);
   return archive(fd, fd2, srcName, size, mtime);
}

void write_EOT(int fd, long size);

void write_EOT_w(HANDLE tarFD, long size)
{
   
   int             fd = _open_osfhandle((intptr_t)tarFD, _O_WRONLY);
   write_EOT(fd, size);
}
*/
import "C"
import (
	"os"
	"unsafe"
)

/*
func Write_EOT(fd *os.File, size int64) {
    C.write_EOT_w((C.HANDLE)(fd.Fd()), (long)(size))
}
*/

func Archive(tar *os.File, sourcefile *os.File, src string, size int64, mtime int64) int {
    srcName := C.CString(src)
    defer C.free(unsafe.Pointer(srcName))

    error := C.archive_w((C.HANDLE)(tar.Fd()), (C.HANDLE)(sourcefile.Fd()), srcName, C.long(size), C.long(mtime))
    return int(error)
}

