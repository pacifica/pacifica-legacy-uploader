// Package archiver provides tar capabilies.
package archiver
import (
	"os"
)

func Archive(tar *os.File, sourcefile *os.File, srcName string, size int64, mtime int64) int {
    error := archive((int)(tar.Fd()), (int)(sourcefile.Fd()), srcName, size, mtime)
    return error 
}

