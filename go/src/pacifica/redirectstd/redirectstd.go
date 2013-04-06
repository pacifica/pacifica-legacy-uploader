package redirectstd

import (
	"os"
)

func RedirectStdErr(file *os.File) error {
	return redirectStdErr(file)
}
