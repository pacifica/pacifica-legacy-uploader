package redirectstd

import (
	"os"
	"syscall"
)

func redirectStdErr(file *os.File) error {
	return syscall.Dup2((int)(file.Fd()), syscall.Stderr)
}
