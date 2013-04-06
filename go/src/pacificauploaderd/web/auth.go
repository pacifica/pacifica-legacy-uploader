package web

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/exec"
	"platform"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"path"
	"pacificauploaderd/common"
)

func authUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic realm=\"Uploader\"")
	w.WriteHeader(http.StatusUnauthorized)
}

func AuthUser(req *http.Request) string {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	auth = strings.Trim(auth[len("Basic "):], " \t")
	userpw, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return ""
	}
	auth = string(userpw)
	sep := strings.Index(auth, ":")
	if sep == -1 {
		return ""
	}
	user := auth[:sep]
	user = strings.TrimSpace(user)
	return user
}

func AuthCheck(w http.ResponseWriter, req *http.Request) bool {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		authUnauthorized(w)
		return false
	}
	auth = strings.Trim(auth, " \t")
	t := auth[0:len("Basic ")]
	t = strings.ToLower(t)
	if t != "basic " {
		authUnauthorized(w)
		return false
	}
	auth = strings.Trim(auth[len("Basic "):], " \t")
	userpw, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		authUnauthorized(w)
		return false
	}
	auth = string(userpw)
	sep := strings.Index(auth, ":")
	if sep == -1 {
		authUnauthorized(w)
		return false
	}
	user := auth[:sep]
	if user == "" {
		authUnauthorized(w)
		return false
	}
	upasswd := auth[sep+1:]
	passwd := authStore.GetPw(user)
	if passwd == "" || passwd != upasswd {
		authUnauthorized(w)
		return false
	}
	return true
}

type AuthStore struct {
	user2pass map[string]string
	mutex     sync.RWMutex
}

func NewAuthStore() *AuthStore {
	return &AuthStore{
		user2pass: make(map[string]string),
	}
}

func (store *AuthStore) GetPw(user string) string {
	store.mutex.RLock()
	passwd := store.user2pass[user]
	store.mutex.RUnlock()
	if passwd == "" {
		_, passwd = store.GetFile(user)
	}
	return passwd
}

func (store *AuthStore) getFinalFilename(user string) string {
	e := base64.StdEncoding.EncodeToString([]byte(user))
	encoded := strings.Replace(e, "/", "-", -1)
	sep := string(os.PathSeparator)
//FIXME filepath join.
	return common.BaseDir + sep + "auth" + sep + "user" + sep + encoded
}

func (store *AuthStore) getProtectedFilename(user string) string {
	e := base64.StdEncoding.EncodeToString([]byte(user))
	encoded := strings.Replace(e, "/", "-", -1)
	sep := string(os.PathSeparator)
	return common.BaseDir + sep + "auth" + sep + "secure" + sep + encoded
}

func (store *AuthStore) GetFile(user string) (string, string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	var upath string
	passwd := store.user2pass[user]
	if passwd != "" {
		upath = store.getFinalFilename(user)
	} else {
		buffer := make([]byte, 33, 33)
		spath := store.getProtectedFilename(user)
		upath = store.getFinalFilename(user)
		file, err := os.OpenFile(spath, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.Printf("Error writing user auth file. %v\n", err)
		} else {
			_, err := rand.Read(buffer)
			if err != nil {
				log.Println("Error on random.", err)
			}
			passwd = base64.StdEncoding.EncodeToString(buffer)
			file.Write([]byte(passwd + "\n"))
			store.user2pass[user] = passwd
			file.Close()
			uid, err := strconv.Atoi(user)
			if err != nil {
				//FIXME
			}
			if common.System {
				switch platform.PlatformGet() {
				case platform.Windows:
					//TODO - remove??? NGT 2/19/2012
					//err = common.Cacls(spath, "/p", "NT AUTHORITY\\SYSTEM:f", user + ":r", "BUILTIN\\Administrators:F")
					user = strings.TrimSpace(user)
					err = common.Cacls(spath, "/p", "NT AUTHORITY\\SYSTEM:f", user+":r", "BUILTIN\\Administrators:F")
				case platform.Linux:
					//FIXME validate username
					cmd := exec.Command("setfacl", "-m", "user:"+strconv.Itoa(uid)+":r", spath)
					err = cmd.Run()
				case platform.Darwin:
					log.Panic("Darwin not yet supported")
				default:
					log.Panic("%v+ not yet supported", runtime.GOOS)
				}
				if err != nil {
					log.Printf("Error %v |%s|\n", err, user)
					return "", ""
				}
			}
			if platform.PlatformGet() == platform.Windows {
				err = os.Remove(upath)
			}
			err = os.Rename(spath, upath)
			if err != nil {
				log.Println("Error %v", err)
			}
		}
	}
	return upath, passwd
}

var authStore = NewAuthStore()

func authHandle(w http.ResponseWriter, req *http.Request) {
	c := strings.Count(req.URL.Path, "/")
	l := len(req.URL.Path)
	if c == 2 || (c == 3 && req.URL.Path[l-1:l] == "/" && req.URL.Path[l-2:l-1] != "/") {
		ss := strings.Split(req.URL.Path, "/")
		user := ss[2]
		if user == "" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			switch platform.PlatformGet() {
			case platform.Windows:
				if strings.Index(user, ":") != -1 {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					path, _ := authStore.GetFile(strings.ToLower(user))
					w.Write([]byte(path + "\n"))
				}
			case platform.Linux:
				path, _ := authStore.GetFile(user)
				w.Write([]byte(path + "\n"))
			case platform.Darwin:
				log.Panic("Darwin not yet supported")
			default:
				log.Panic("%v+ not yet supported", runtime.GOOS)
			}
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func authInit() {
	os.MkdirAll(path.Join(common.BaseDir, "auth"), 0755)
	err := os.RemoveAll(path.Join(common.BaseDir, "auth", "secure"))
	if err != nil {
		log.Printf("Failed to clean out the secure directory. %v\n", err)
		os.Exit(-1)
	}
	err = os.RemoveAll(path.Join(common.BaseDir, "auth", "user"))
	if err != nil {
		log.Printf("Failed to clean out the user directory. %v\n", err)
		os.Exit(-1)
	}
	os.MkdirAll(path.Join(common.BaseDir, "auth", "user"), 0755)
	os.MkdirAll(path.Join(common.BaseDir, "auth", "secure"), 0700)
	if common.System && platform.PlatformGet() == platform.Windows {
		err = common.Cacls(path.Join(common.BaseDir, "auth", "secure"), "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
		err = common.Cacls(path.Join(common.BaseDir, "auth", "user"), "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Users:r", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls on auth user %v\n", err)
			os.Exit(-1)
		}
	}
}
