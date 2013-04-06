package common

import (
	"errors"
	"log"
	"pacifica/pipepair"
	userdrpc "pacificauploaderuserd/rpc"
	"net/rpc"
	"os"
	"os/exec"
	"platform"
	"sync"
	"time"
)

var userd_fallback *rpc.Client
var userd map[string]*rpc.Client
var mutex sync.Mutex

func userdForUserGet(user string) (*rpc.Client, error) {
	if System && platform.PlatformGet() == platform.Linux {
		mutex.Lock()
		defer mutex.Unlock()
		if userd[user] != nil {
			return userd[user], nil
		}
		tmpuserd, err := userdSpawn(user)
		if err != nil {
			return nil, err
		}
		userd[user] = tmpuserd
		return tmpuserd, nil
	}
	if userd_fallback == nil {
		return nil, errors.New("Bad userd")
	}
	return userd_fallback, nil
}

func UserAccess(user string, fullpath string) (bool, error) {
	var reply bool
	client, err := userdForUserGet(user)
	if err != nil {
		return false, err
	}
	if err := client.Call(userdrpc.ACCESS, userdrpc.AccessArgs{Path: fullpath}, &reply); err != nil {
		return false, err
	}
	return reply, nil
}

func UserBundleFile(user string, bundle_path string, local_path string, name string) (bool, string, userdrpc.BundleFileError, time.Time, error) {
	var reply userdrpc.BundleFileResult
	client, err := userdForUserGet(user)
	if err != nil {
		return false, "", userdrpc.BundleFileError_UNKNOWN, time.Now(), err
	}
	if err := client.Call(userdrpc.BUNDLEFILE, userdrpc.BundleFileArgs{BundlePath: bundle_path, LocalPath: local_path, Name: name}, &reply); err != nil {
		return false, "", userdrpc.BundleFileError_UNKNOWN, time.Now(), err
	}
	return reply.Retval, reply.Sha1, reply.Error, reply.Mtime, nil
}

const (
	BUFFSIZE int = 1024
)

func userdSpawn(user string) (*rpc.Client, error) {
	var buffer [BUFFSIZE]byte
	//FIXME error handling.
	userd_path := UserdPathGet()
	var cmd *exec.Cmd
	if user != "" {
		userswitcher_path := UserSwitcherPathGet()
		cmd = exec.Command(userswitcher_path, "-u", user, userd_path)
	} else {
		cmd = exec.Command(userd_path)
	}

	//TODO - remove NGT 7/13/2012
	/* else if System && platform.PlatformGet() == platform.Windows {
		userswitcher_path := UserSwitcherPathGet()
		creds := filepath.Join(BaseDir, "priv", "localservice.cred");
		log.Printf("Execing \"%s\" -u \"Pacifica Uploader\" -p \"%s\" \"%s\"\n", userswitcher_path, creds, userd_path)
		cmd = exec.Command(userswitcher_path, "-u", "Pacifica Uploader", "-p", creds, userd_path)
	}*/
	pipe_write, _ := cmd.StdinPipe()
	pipe_read, _ := cmd.StdoutPipe()
	pipe_err, _ := cmd.StderrPipe()
	pipepair := pipepair.PipePair{In: pipe_read, Out: pipe_write}
	_ = cmd.Start()
	go func() {
		for {
			num, err := pipe_err.Read(buffer[:])
			if err != nil || num < 0 {
				break
			}
			log.Print(string(buffer[0:num]))
		}
		e := cmd.Wait()
		if e != nil {
			log.Print("Userd exited with %v\n", e)
		}
	}()
	return rpc.NewClient(pipepair), nil
}

func userdInit() {
	userd = make(map[string]*rpc.Client)
	if System == false || platform.PlatformGet() != platform.Linux {
		var err error
		userd_fallback, err = userdSpawn("")
		if err != nil {
			log.Printf("Failed to get default userd. %v\n", err)
			os.Exit(1)
		}
	}
}
