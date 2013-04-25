package upload

import (
	"io"
	"os"
	"fmt"
	"log"
	"time"
	"platform"
	"net/http"
	"io/ioutil"
	"path/filepath"
	"pacificauploaderd/web"
	"pacificauploaderd/common"
)

import (
	pacificaauth "pacifica/auth"
)

type CredsManager struct {
	userCreds map[string]*pacificaauth.Auth
	outage map[string]*time.Time
}

const (
	BUFFSIZE int = 1024
	MAXCRED int = 1024 * 1024
)

func passcredsHandle(cm *CredsManager, w http.ResponseWriter, req *http.Request) {
	if web.AuthCheck(w, req) {
		if req.Method != "PUT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
			file, err := ioutil.TempFile(filepath.Join(common.BaseDir, "auth", "tmpcreds"), "creds")
			if err != nil {
//FIXME what to return here?
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var buffer [BUFFSIZE]byte
			total := 0
			for {
				num, err := req.Body.Read(buffer[:])
				if (err != nil && err != io.EOF) || num < 0 {
					w.WriteHeader(http.StatusBadRequest)
					log.Printf("Failed to read during pass creds. %v %v", file.Name(), err)
					file.Close()
					os.Remove(file.Name())
					
					return
				}
				if num == 0 {
					break
				}
				total += num
				if total > MAXCRED {
					w.WriteHeader(http.StatusBadRequest)
					log.Printf("Cred call too big.")
					file.Close()
					os.Remove(file.Name())
					return
				}
				_, err = file.Write(buffer[0:num])
				if err != nil {
//FIXME what to return here?
					w.WriteHeader(http.StatusBadRequest)
					log.Printf("Failed to write to temp cred file. %v\n", err)
					file.Close()
					os.Remove(file.Name())
					return
				}
			}
			_, err = file.Seek(0, 0)
			if err != nil {
//FIXME what to return here?
				w.WriteHeader(http.StatusBadRequest)
				os.Remove(file.Name())
				return
			}
			auth := pacificaauth.NewAuth(file)
			file.Close()
			log.Println("testauth:", auth.Services["testauth"])
			client := auth.NewClient()
			r, err := client.Get(auth.Services["testauth"])
			if err != nil {
				log.Println(err)
				os.Remove(file.Name())
				return
			}
			log.Println(r.StatusCode)
			user := web.AuthUser(req)
			upath := filepath.Join(common.BaseDir, "auth", "creds", common.FsEncodeUser(user))
			if platform.PlatformGet() == platform.Windows {
				err = os.Remove(upath)
			}
			err = os.Rename(file.Name(), upath)
			if err != nil {
				log.Println(err)
				os.Remove(file.Name())
				return
			}
			if cm.userCreds[user] != nil {
				oldclient := cm.userCreds[user].NewClient()
				r, err = oldclient.Get(auth.Services["logout"])
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("Logged out %v: %v\n", user, r.StatusCode)
				}
			}
			cm.userCreds[user] = auth
			cm.outage[user] = nil
		}
	}
}

func statusXmlHandle(cm *CredsManager, w http.ResponseWriter, req *http.Request) {
	if web.AuthCheck(w, req) {
		user := web.AuthUser(req)
		var state string
		if cm.userCreds[user] == nil {
			state = "auth"
		} else {
			state = bm.BundleUserState(user)
		}
		_, err := w.Write([]byte("<?xml version=\"1.0\"?>\n<pacifica_uploader><status>" + state + "</status></pacifica_uploader>"))
		if err != nil {
			fmt.Fprintf(w, "%v", err)
		}
	}
}

func credsInit() (*CredsManager) {
	cm := &CredsManager{}
	cm.userCreds = make(map[string]*pacificaauth.Auth)
	cm.outage = make(map[string]*time.Time)
	web.ServMux.HandleFunc("/passcreds/", func(w http.ResponseWriter, req *http.Request) { passcredsHandle(cm, w, req) })
	web.ServMux.HandleFunc("/status/xml/", func(w http.ResponseWriter, req *http.Request) { statusXmlHandle(cm, w, req) })
	os.MkdirAll(filepath.Join(common.BaseDir, "auth", "creds"), 0700)
	err := os.RemoveAll(filepath.Join(common.BaseDir, "auth", "tmpcreds"))
	if err != nil {
		log.Printf("Failed to clean out the tmpcreds directory. %v\n", err)
		os.Exit(-1)
	}
	os.MkdirAll(filepath.Join(common.BaseDir, "auth", "tmpcreds"), 0700)
	if common.System && platform.PlatformGet() == platform.Windows {
		err = common.Cacls(filepath.Join(common.BaseDir, "auth", "creds"), "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
		err = common.Cacls(filepath.Join(common.BaseDir, "auth", "tmpcreds"), "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
	}
	creds, err := filepath.Glob(filepath.Join(common.BaseDir, "auth", "creds", "*"))
	if err == nil {
		for _, credfile := range creds {
			user := common.FsDecodeUser(credfile)
			if user != "" {
				file, err := os.Open(credfile)
				if err == nil {
					auth := pacificaauth.NewAuth(file)
					if auth != nil {
						log.Printf("Found cred file. %v %v\n", credfile, user)
						cm.userCreds[user] = auth
					}
				}
			}
		}
	}
	return cm
}
