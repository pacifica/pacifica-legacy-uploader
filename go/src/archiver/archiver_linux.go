// Package archiver provides tar capabilies.
package archiver
import (
	"os"
)


func Write_EOT(fd *os.File, size int64) {
    write_EOT((int)(fd.Fd()), size)
}

func Archive(tar *os.File, sourcefile *os.File, srcName string, size int64, mtime int64) int {
    error := archive((int)(tar.Fd()), (int)(sourcefile.Fd()), srcName, size, mtime)
    return error 
}

