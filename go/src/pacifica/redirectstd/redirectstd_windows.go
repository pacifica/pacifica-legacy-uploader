package redirectstd

/*
#include <windows.h>
#include <winbase.h>

int redirectStdErr_go(HANDLE h)
{
	return SetStdHandle(STD_ERROR_HANDLE, h);
}

*/
import "C"
import (
	"os"
	"errors"
)

func redirectStdErr(file *os.File) error {
	b := C.redirectStdErr_go((C.HANDLE)(file.Fd()))
	if b == 0 {
		return errors.New("Error")
	}
	return nil

}
