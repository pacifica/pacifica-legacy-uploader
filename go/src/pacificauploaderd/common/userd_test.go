package common

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"testing"
)

func Test_FakeUserCannotAccessFile(t *testing.T) {
	fmt.Printf("FakeUserCannotAccessfile")
	fakeUser := "PNL\a1sdfata"
	file, _ := exec.LookPath(os.Args[0]) // [1]
	fmt.Println("file:", file)
	cwd, _ := os.Getwd()
	if _, err := UserAccess(fakeUser, cwd); err == nil {
		t.Error("Fake user: %v was allowed access to %v", fakeUser, cwd)
	}
}

func Test_CurrentUserCanAccessFile(t *testing.T) {
	fmt.Printf("CurrentUserCanAccessFile")

	u, err := user.Current()
	if err != nil {
		t.Error("Couldn't get current user\n%v", err)
		return
	}
	file, _ := exec.LookPath(os.Args[0]) // [1]
	fmt.Println("file:", file)
	cwd, _ := os.Getwd()
	if _, err := UserAccess(u.Username, cwd); err != nil {
		t.Error("User: %v couldn't access %v\nError: %v", u.Username, cwd, err)
		return
	}
}
