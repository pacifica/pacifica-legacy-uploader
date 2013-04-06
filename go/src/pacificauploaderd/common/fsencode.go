package common

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
)

func FsEncodeUser(user string) string {
	e := base64.StdEncoding.EncodeToString([]byte(user))
	encoded := strings.Replace(e, string(os.PathSeparator), "-", -1)
	return encoded
}

func FsDecodeUser(filename string) string {
	f := filepath.Base(filename)
	ext := filepath.Ext(f)
	userName := f[0 : len(f)-len(ext)]
	userName = strings.Replace(userName, "-", string(os.PathSeparator), -1)
	t, err := base64.StdEncoding.DecodeString(userName)
	if err != nil {
		return ""
	}
	userName = string(t)
	return userName
}
