package getmodule

/*
#cgo LDFLAGS: -lnetapi32
#include <windows.h>
#include <winbase.h>
#include <lm.h>

int pacifica_win32_get_machine_name(WCHAR *buffer, int len)
{
	int res = -1;
	DWORD level = 102;
	NET_API_STATUS status;
	LPWKSTA_INFO_102 buf;
	status = NetWkstaGetInfo(NULL, level, (LPBYTE*)(&buf));
	if(status == NERR_Success)
	{
		if(wcslen(buf->wki102_computername) <= len)
		{
			wcscpy(buffer, buf->wki102_computername);
			res = wcslen(buf->wki102_computername);
		}
		NetApiBufferFree(buf);
	}
	return res;
}

*/
import "C"
import (
	"errors"
	"unicode/utf16"
	//	"path"
)

func GetModuleFileName() (string, error) {
	buffer := make([]uint16, 512)
	l := C.GetModuleFileNameW(nil, (*C.WCHAR)(&buffer[0]), (C.DWORD)(uint32(len(buffer))))
	if l < 0 {
		return "", errors.New("Error")
	}
	return string(utf16.Decode(buffer[0:l])), nil

}

func GetMachineName() (string, error) {
	buffer := make([]uint16, 512)
	l := C.pacifica_win32_get_machine_name((*C.WCHAR)(&buffer[0]), (C.int)(len(buffer)))
	if l < 0 {
		return "", errors.New("Error")
	}
	return string(utf16.Decode(buffer[0:l])), nil
}

/*
func GetModuleDirName() (string, error) {
	var str string
	if str, err := GetModuleFileName(); err != nil {
		return str, err
	}
	return path.Dir(str), nil
}
*/
