package auto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"pacificauploaderd/common"
	"pacificauploaderd/web"
	"net/http"
	"os"
	"path/filepath"
	"platform"
	"strings"
	"text/template"
)

const (
	_USER_CONFIG_EXT string = ".json"
)

type configOp int

const (
	setConfig configOp = iota
	getConfig
)

var (
	configTemplate *template.Template
	configChannel  chan configRequest
	allConfigs     map[string]*UserConfig = make(map[string]*UserConfig)
)

type configRequest struct {
	user string
	c    UserConfig
	op   configOp
	ret  chan UserConfig
}

type MetadataGroup struct {
	Pattern string
	Value   string
}

type MetadataExtract struct {
	Pattern string
	Group   []MetadataGroup
}

type StaticGroup struct {
	Type string
	Name string
}

type RenamePattern struct {
	Pattern string
	Value   string
}

type ReverseLookupEntry struct {
	User   string
	Prefix string
	Rule   *WatchRule
}

type WatchRule struct {
	Name            string
	Paths           []string
	StaticMetadata  []StaticGroup
	ExcludePatterns []string
	RenamePatterns  []RenamePattern
	MetadataPattern []MetadataExtract
	AutoSubmit      bool
	AutoDelete      bool
	Atomic          bool
}

type UserConfig struct {
	Rules []*WatchRule
}

func (self *UserConfig) Save(filename string) error {
	j, err := json.Marshal(self)
	if err != nil {
		return err
	}

	buff := new(bytes.Buffer)
	json.Indent(buff, j, "", "\t")

	file, err := os.Create(filename + ".new")
	if err != nil {
		return err
	}

	_, err = file.Write(buff.Bytes())
	if err != nil {
		return err
	}
	file.Close()

	if platform.PlatformGet() == platform.Windows {
		os.Remove(filename)
	}
	if err := os.Rename(filename+".new", filename); err != nil {
		log.Printf("Failed to rename %s, error %v", filename, err)
		return err
	}

	return nil
}

func ReadConfig(filename string) (*UserConfig, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var c UserConfig
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func ConfigRun() {
	go func() {
		for {
			r := <-configChannel
			switch r.op {
			case setConfig:
				e := base64.StdEncoding.EncodeToString([]byte(r.user))
				encoded := strings.Replace(e, string(os.PathSeparator), "-", -1)
				log.Printf("set %s %s", r.user, encoded)
				r.c.Save(filepath.Join(common.BaseDir, "config", encoded+_USER_CONFIG_EXT))
				allConfigs[r.user] = &r.c
				m.updatePaths(&allConfigs)
			case getConfig:
				if uconfig := allConfigs[r.user]; uconfig == nil {
					r.ret <- UserConfig{Rules: nil}
				} else {
					r.ret <- *uconfig
				}
			}
			log.Printf("r.c = %+v", r.c)
		}
	}()
}

func configHandle(w http.ResponseWriter, req *http.Request) {
	if web.AuthCheck(w, req) {
		user := web.AuthUser(req)
		if req.Method == "PUT" {
			dec := json.NewDecoder(req.Body)
			for {
				var uc UserConfig
				if err := dec.Decode(&uc); err == io.EOF {
					break
				} else if err != nil {
					log.Print("Error %v", err)
					break
				}
				configChannel <- configRequest{c: uc, op: setConfig, user: user}
			}
		}
		r := configRequest{op: getConfig, ret: make(chan UserConfig), user: user}
		configChannel <- r
		j, err := json.Marshal(<-r.ret)
		if err != nil {
			fmt.Fprintf(w, "%v\n", err)
			return
		}

		buff := new(bytes.Buffer)
		json.Indent(buff, j, "", "\t")

		if err != nil {
			log.Printf("%v\n", err)
		}
		fmt.Fprintf(w, "%s", buff)
	}
}

func configInit() {
	t, err := template.ParseFiles(filepath.Join(common.UiDirGet(), "config.html"))
	if err != nil {
		log.Printf("config.html parse failed %v\n", err)
		return
	} else {
		configTemplate = t
	}

	configChannel = make(chan configRequest)

	ConfigRun()

	web.ServMux.Handle("/config/", http.RedirectHandler("/ui/config.html", http.StatusMovedPermanently))
	web.ServMux.HandleFunc("/config/all/", configHandle)

	os.MkdirAll(filepath.Join(common.BaseDir, "config"), 0700)
	if common.System && platform.PlatformGet() == platform.Windows {
		err = common.Cacls(filepath.Join(common.BaseDir, "config"), "/p", "NT AUTHORITY\\SYSTEM:f", "BUILTIN\\Administrators:F")
		if err != nil {
			log.Printf("Failed to run cacls %v\n", err)
			os.Exit(-1)
		}
	}

	processConfig()
	m.updatePaths(&allConfigs)
}

func processConfig() {
	configs, err := filepath.Glob(filepath.Join(common.BaseDir, "config", "*"+_USER_CONFIG_EXT))
	if err != nil {
		log.Panic("Failed to get configuration files %v", err)
	}

	for _, v := range configs {
		user := getUserFromFile(v)
		if user == "" {
			continue
		}
		log.Printf("Found user %s", user)
		c, err := ReadConfig(v)
		if user == "" || err != nil {
			log.Printf("Failed to read config file %s %v", v, err)
			continue
		}
		allConfigs[user] = c
	}
}

func getUserFromFile(filename string) string {
	_, f := filepath.Split(filename)
	ext := filepath.Ext(filename)
	userName := f[0 : len(f)-len(ext)]
	userName = strings.Replace(userName, "-", string(os.PathSeparator), -1)
	t, err := base64.StdEncoding.DecodeString(userName)
	if err != nil {
		return ""
	}
	userName = string(t)
	return userName
}

func getWatchRule(userName, ruleName string) *WatchRule {
	uc, ok := allConfigs[userName]
	if !ok {
		return nil
	}
	for _, v := range uc.Rules {
		if v.Name == ruleName {
			return v
		}
	}
	return nil
}
