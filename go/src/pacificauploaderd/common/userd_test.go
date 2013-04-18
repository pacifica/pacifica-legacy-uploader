package common

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func Test_FakeUserCannotAccessFile(t *testing.T) {
	fmt.Printf("FakeUserCannotAccessfile")
	fakeUser := "PNL\a1sdfata"
	file, _ := exec.LookPath(os.Args[0]) // [1]
	println("file:", file)
	cwd, err := os.Getwd(file)
	if _, err := UserAccess(fakeUser, cwd); err == nil {
		t.Error("Fake user: %v was allowed access to %v", fakeUser, cwd)
	}
	t.Error("Just making sure it is here!")
}
