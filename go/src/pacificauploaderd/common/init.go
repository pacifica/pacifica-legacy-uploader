package common

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"platform"
	"strings"
)

var (
	Devel    bool
	System   bool
	BaseDir  string
	StateDir string
	Uuid     string
)

func uuidInit() {
	file, err := os.Open(filepath.Join(BaseDir, "state", "uuid"))
	if err == nil {
		defer file.Close()
		r := bufio.NewReaderSize(file, 1024)
		line, isPrefix, err := r.ReadLine()
		for err == nil && !isPrefix {
			Uuid = fmt.Sprintf("%s", line)
			return
		}
	}
	cmd := exec.Command(UuidgenPathGet())
	pipe_read, err := cmd.StdoutPipe()
	for err != nil {
		log.Printf("Failed create uuidgen pipe.\n")
		os.Exit(-1)
	}
	err = cmd.Start()
	for err != nil {
		log.Printf("Failed run uuidgen.\n")
		os.Exit(-1)
	}
	r := bufio.NewReaderSize(pipe_read, 1024)
	line, isPrefix, err := r.ReadLine()
	for err != nil || isPrefix {
		log.Printf("Failed to read from uuidgen pipe. %v\n", err)
		os.Exit(-1)
	}
	cmd.Wait()
	Uuid = fmt.Sprintf("%s", line)
	file, err = os.Create(filepath.Join(BaseDir, "state", "uuid"))
	if err != nil {
		log.Printf("Failed to create uuid file. %v\n", err)
		os.Exit(-1)
	}
	file.Write(line)
	file.Close()
}

func Init() {
	log.Println("Common subsystem init.")

	if System && BaseDir == DefaultBaseDirGet() {
		BaseDir = DefaultSystemBaseDirGet()
	}
	oldname := strings.Replace(BaseDir, "pacifica", "myemsl", -1)
	oldname = strings.Replace(oldname, "Pacifica", "MyEMSL", -1)
//FIXME Old migration code. Remove someday.
	fi, err := os.Stat(oldname)
	if err == nil && fi.IsDir() {
		log.Printf("This system needs to be migrated. Bailing.\n")
		os.Exit(-1)
	}
//FIXME End of old migration code.
	os.MkdirAll(BaseDir, 0755)
	setupLogger(LogDirGet())
	StateDir = filepath.Join(BaseDir, "state")
	os.MkdirAll(StateDir, 0700)
	os.MkdirAll(filepath.Join(BaseDir, "auth"), 0755)
	if System && platform.PlatformGet() == platform.Windows {
		err := Cacls(BaseDir, "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Users:r", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
	}
	if System && platform.PlatformGet() == platform.Windows {
		err := Cacls(StateDir, "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
	}
	_, err = os.Stat(BaseDir)
	if err != nil {
		log.Printf("The specified basedir is not valid, %v\n", err)
	}
	VersionCheckerInit()
	userdInit()
	uuidInit()
	log.Printf("Got uuid %s\n", Uuid)
}
