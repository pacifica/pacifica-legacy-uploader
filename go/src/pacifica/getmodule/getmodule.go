package getmodule

import (
	"path/filepath"
)

func GetModuleDirName() (string, error) {
	str, err := GetModuleFileName()
	if err != nil {
		return str, err
	}
	return filepath.Dir(str), nil
}
