package common

import (
	"log"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

const (
	_DEFAULT_USER_BASEDIR   string = "/.pacifica/uploader"
	_DEFAULT_SYSTEM_BASEDIR string = "/var/lib/pacifica/uploader"
	_LOG_FILENAME           string = "pacifica_uploader.log"
)

func UserdDefaultUsername() string {
	return ""
}

func Cacls(elements ...string) error {
	return nil
}

func UiDirGet() string {
	return "/usr/share/pacifica/uploader/ui"
}

func LogDirGet() string {
	if System {
		return "/var/log"
	}
	return BaseDir
}

func UserdPathGet() string {
	return "/usr/libexec/pacifica/uploader/userd"
}

func UserSwitcherPathGet() string {
	return "/usr/libexec/pacifica/userswitcher"
}

func UuidgenPathGet() string {
	return "/usr/libexec/pacifica/uuidgen"
}

func DefaultBaseDirGet() string {
	uid := syscall.Getuid()
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		log.Printf("Error looking up user, %v\n", err)
		os.Exit(-2)
	}
	return u.HomeDir + _DEFAULT_USER_BASEDIR
}

func DefaultSystemBaseDirGet() string {
	return _DEFAULT_SYSTEM_BASEDIR
}
