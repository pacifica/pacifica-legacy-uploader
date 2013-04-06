package common

/*
#include <windows.h>
#include <winbase.h>
#include <winreg.h>

int pacifica_service_url_get(char *value, CHAR *string, DWORD size) {
	LONG l;
	HKEY key;
	l = RegOpenKeyEx(HKEY_LOCAL_MACHINE, "Software\\Pacifica\\Pacifica Auth\\Url", 0, KEY_READ, &key);
	if(l != ERROR_SUCCESS)
	{
		return 1;
	}
	l = RegQueryValueEx(key, value, NULL, NULL, (LPBYTE)string, &size);
	RegCloseKey(key);
	if(l == ERROR_SUCCESS)
	{
		return 0;
	}
	return 1;
}
*/
import "C"

import (
	"crypto/tls"
	"easyhttp"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

const (
	versionCheckFrequency = 24 * time.Hour
)

var (
	OutOfDate         bool   = false
	OutOfDateUrl      string = ""
	OutOfDateChecking bool   = false
	UpdateDownloaded  bool   = false
	UpdatePath        string = ""
	versionCheckMutex sync.Mutex
)

//FIXME some of this code belongs in a general pacifica go library. Probably with auth.
type Service struct {
	Name     string `xml:"name,attr"`
	Location string `xml:"location,attr"`
}

type Result struct {
	XMLName  xml.Name  `xml:"myemsl"`
	Prefix   string    `xml:"prefix"`
	Services []Service `xml:"services>service"`
}

func periodicVersionCheck(serviceUrl string, insecure bool) {
	for {
		log.Println("Checking for new version...")
		versionCheck(serviceUrl, insecure)
		if OutOfDate && !UpdateDownloaded {
			log.Printf("Attempting to download new version from %v", OutOfDateUrl)
			saveAs := filepath.Join(BaseDir, "update", "pacificauploadersetup.exe")
			resultChan := easyhttp.DownloadAsync(OutOfDateUrl, saveAs)
			dr := <-resultChan
			if dr.Err != nil {
				log.Printf("Update download failed with error %v", dr.Err)
				UpdateDownloaded = false
				continue
			}
			UpdateDownloaded = true
			UpdatePath = dr.Path
			log.Printf("Downloaded update complete, saved to %v", UpdatePath)
		}
		log.Printf("Pausing version check for %v", versionCheckFrequency)
		<-time.After(versionCheckFrequency)
	}
}

func versionCheck(serviceUrl string, insecure bool) {
	versionCheckMutex.Lock()
	defer versionCheckMutex.Unlock()

	OutOfDateChecking = true
	defer func() { OutOfDateChecking = false }()

	client := &http.Client{}
	if insecure == true {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		client.Transport = tr
	}
	r, err := client.Get(serviceUrl)
	if err != nil {
		log.Printf("Failed to get http client. %v\n", err)
		return
	}
	defer r.Body.Close()
	log.Println("Getting service xml done.")
	log.Println(r.StatusCode)
	if r.StatusCode == 200 {
		v := Result{}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read status xml. %v\n", err)
			return
		}
		err = xml.Unmarshal(b, &v)
		if err != nil {
			log.Printf("Failed to parse status xml. %v\n", err)
			return
		}
		clientUrl := ""
		for id, _ := range v.Services {
			if v.Services[id].Location[0] == '/' {
				v.Services[id].Location = v.Prefix + v.Services[id].Location
			}
			if v.Services[id].Name == "client" {
				clientUrl = v.Services[id].Location
			}
		}
		if clientUrl != "" {
			url := clientUrl + "windows/upgrading/from-" + VERSION + "/pacificauploadersetup.exe"
			log.Printf("Trying for %s\n", url)
			r, err := client.Head(url)
			if err != nil {
				log.Printf("Failed to head with http client. %v\n", err)
				return
			}
			log.Printf("Upgrades status %v\n", r.StatusCode)
			if r.StatusCode == 200 {
				log.Printf("Upgrade found!\n")
				OutOfDate = true
				OutOfDateUrl = url
			}
		}
	}
}

func VersionCheckerInit() {
	log.Println("Entering VersionCheckerInit")
	defer log.Println("Leaving VersionCheckInit")
	if Devel == true {
		return
	}
	serviceUrl, err := PacificaServiceUrlGet()
	if err != nil {
		log.Printf("Failed to get service url from the registry. %v\n", err)
		return
	}
	insecure, err := PacificaServiceInsecureGet()
	if err != nil {
		log.Printf("Failed to get service insecure from the registry. %v\n", err)
		return
	}
	log.Printf("Got service url %s %v\n", serviceUrl, insecure)
	go periodicVersionCheck(serviceUrl, insecure)
}

func PacificaServiceUrlGet() (string, error) {
	return pacificaServiceUrlGet("services")
}

func PacificaServiceInsecureGet() (bool, error) {
	value, err := pacificaServiceUrlGet("insecure")
	if err != nil {
		return false, err
	}
	return value == "True", nil
}

func pacificaServiceUrlGet(value string) (string, error) {
	buffer := make([]int8, 512)
	s := C.CString(value)
	defer C.free(unsafe.Pointer(s))
	l := C.pacifica_service_url_get(s, (*C.CHAR)(&buffer[0]), (C.DWORD)(uint32(len(buffer))))
	if l != 0 {
		return "", errors.New("Error")
	}
	retval := C.GoString((*C.char)(&buffer[0]))
	return retval, nil
}
