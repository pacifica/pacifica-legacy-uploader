package common

import (
	"fmt"
	"log"
	"pacifica/getmodule"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	_UPLOADER_DIR                    = "\\Pacifica\\Uploader"
	_DEFAULT_USER_BASEDIR_OLD string = "\\Local Settings\\Application Data"
	_DEFAULT_USER_BASEDIR_NEW string = "\\AppData\\Local"
	_LOG_FILENAME             string = "pacificauploaderd.log"
)

var (
	_BASEDIR_USER string
)

func UserdDefaultUsername() string {
	name, err := getmodule.GetMachineName()
	log.Printf("Machine Name: %v %v\n", name, err)
	//TODO - remove NGT 7/13/2012
	//return name + "\\Pacifica Uploader"
	return name + "\\SYSTEM"
}

func Cacls(elements ...string) error {
	cmdStr := "cacls"
	for _, s := range elements {
		cmdStr += " " + s
	}
	log.Printf("command ready to be executed %v", cmdStr)

	cmd := exec.Command("cacls", elements...)
	pipe, err := cmd.StdinPipe()
	pipe.Write([]byte("Y\n"))
	err = cmd.Run()
	return err
}

func baseDirGetUser() string {
	userprofile := os.Getenv("USERPROFILE")
	if _BASEDIR_USER != "" {
		return _BASEDIR_USER + "\\Pacifica\\Uploader"
	}
	_BASEDIR_USER = userprofile + _DEFAULT_USER_BASEDIR_NEW
	_, err := os.Stat(_BASEDIR_USER)
	if err != nil {
		_BASEDIR_USER = userprofile + _DEFAULT_USER_BASEDIR_OLD
	}
	return _BASEDIR_USER + "\\Pacifica\\Uploader"
}

func UiDirGet() string {
	path, err := getmodule.GetModuleDirName()
	if err != nil {
		fmt.Println("Failed to get installed dir.", err)
		return "."
	}
	if Devel {
		path = filepath.Join(path, "..\\go\\src\\pacificauploaderd")
	}
	path = filepath.Join(path, "UI")
	log.Printf("UiDirGet returning %s", path)
	return path
}

func LogDirGet() string {
	return BaseDir
}

func UserdPathGet() string {
	path, err := getmodule.GetModuleDirName()
	if err != nil {
		fmt.Println("Failed to get installed dir.", err)
		return "."
	}
	path = filepath.Join(path, "pacificauploaderuserd.exe")
	log.Printf("UserdPathGet returning %s", path)
	return path
}

func UserSwitcherPathGet() string {
	path, err := getmodule.GetModuleDirName()
	if err != nil {
		fmt.Println("Failed to get installed dir.", err)
		return "."
	}
	path = filepath.Join(path, "pacificauploaderuserswitcher.exe")
	log.Printf("UserdPathGet returning %s", path)
	return path
}

func UuidgenPathGet() string {
	path, err := getmodule.GetModuleDirName()
	if err != nil {
		fmt.Println("Failed to get installed dir.", err)
		return "."
	}
	path = filepath.Join(path, "pacificauuidgen.exe")
	log.Printf("UuidgenPathGet returning %s", path)
	return path
}

func DefaultBaseDirGet() string {
	return baseDirGetUser()
}

func DefaultSystemBaseDirGet() string {
	dir := os.Getenv("ALLUSERSPROFILE")
	if dir == "" {
		log.Panic("ALLUSERSPROFILE environment variable not defined, this is unexpected.")
	}
	if dir != "C:\\ProgramData" {
		dir = filepath.Join(dir, "Application Data")
	}
	return filepath.Join(dir, "Pacifica\\Uploader")
}
