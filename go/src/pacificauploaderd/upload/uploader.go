package upload

import (
	"io"
	"os"
	"log"
	"sync"
	"time"
	"bufio"
	"strings"
	"net/url"
	"net/http"
)

type uploader struct {
	wake *sync.Cond
	bm *BundleManager
	cm *CredsManager
}

func uploaderNew(bm *BundleManager, cm *CredsManager) *uploader {
	self := &uploader{bm: bm, cm: cm, wake: sync.NewCond(&sync.Mutex{})}
	err := bm.BundleStateChangeWatchSet(BundleState_ToUpload, func(user string, bundle_id int, state BundleState) { self.Wakeup() })
	if err != nil {
		log.Printf("Failed to register wakeup callback with the bundle manager.")
		return nil
	}
	go func() {
		for {
//FIXME make this configurable
			time.Sleep(5 * time.Minute)
			self.Wakeup()
		}
	} ()
	go func() {
		for {
			found := false
			ids, err := self.bm.bundleIdsForStateGet(BundleState_ToUpload)
			if err == nil && ids != nil {
				log.Printf("Found %v entries to upload.", len(ids))
				for user, list := range ids {
					usercreds_bad := false
					in_outage := false
					creds := self.cm.userCreds[user]
					if creds == nil {
						log.Printf("No creds for user %v\n", user)
						continue
					}
					client := creds.NewClient()
					for _, id := range list {
						bundle_file, err := bm.BundleStringGet(user, id, "bundle_location")
						if err != nil {
							log.Printf("Failed to get bundle location from %v. %v\n", id, err)
//FIXME flag bundle for error?
							continue
						}
						bundle_handle, err := os.Open(bundle_file)
						if err != nil {
							log.Printf("Failed to open bundle from %v. %v\n", id, err)
//FIXME flag bundle for error?
							continue
						}
						log.Printf("Got bundles to upload. %v %v %v\n", user, id, bundle_file)
						log.Println("Calling preallocate.")
						preallocate := creds.Services["preallocate"]
						if preallocate == "" {
							log.Printf("Users creds are bad. Invalidating. %v\n", user)
							self.cm.userCreds[user] = nil
							usercreds_bad = true
							break
						}
						r, err := client.Get(preallocate)
						if err != nil {
							log.Printf("Failed to talk to server.")
							continue
						}
						log.Printf("preallocate: %v\n", r.StatusCode)
						if r.StatusCode == 401 {
							log.Printf("Users creds are too old. Invalidating. %v\n", user)
							self.cm.userCreds[user] = nil
							usercreds_bad = true
							break
						}
						if r.StatusCode != 200 {
							in_outage = true
							log.Printf("Got strange status code from preallocate. %v", int(r.StatusCode))
							break
						}
						readbuf := bufio.NewReader(r.Body)
						server := ""
						location := ""
						outage := ""
						for {
							data, isPrefix, err := readbuf.ReadLine();
							if(isPrefix) {
								log.Println("Line too long");
								break
							} else if(err == io.EOF) {
								break;
							} else if(err != nil) {
								log.Println(err)
								break
							}
							str := string(data)
							if(strings.HasPrefix(str, "Server:")) {
								server = strings.TrimSpace(str[len("Server:"):])
							} else if(strings.HasPrefix(str, "Location:")) {
								location = strings.TrimSpace(str[len("Location:"):])
							} else if(strings.HasPrefix(str, "Outage:")) {
								outage = strings.TrimSpace(str[len("Outage:"):])
							}
						}
						if outage != "" {
							in_outage = true
							break
						}
						if server == "" || location == "" {
							in_outage = true
							log.Printf("Failed to get good information from preallocate and outage not seen.")
							break
						}
						log.Printf("Uploading to %v %v.\n", server, location)
//FIXME configurable protocol
						surl := "https://" + server + location
						req, err := http.NewRequest("PUT", surl, bundle_handle)
						if err != nil {
							log.Printf("Failed to make upload request %v.\n", req)
							continue	
						}
						turl, err := url.Parse(surl)
						if err != nil {
							log.Printf("Failed to parse url.\n")
							continue
						}
						for _, cookie := range client.Jar.Cookies(turl) {
							req.AddCookie(cookie)
						}
						resp, err := client.Do(req)
						if err != nil || resp.StatusCode != 204 {
//FIXME
							log.Printf("Failed to put bundle. %v.\n", err)
							continue	
						}
						log.Println("Calling finalize.")
//FIXME protocol
						finalize := "https://" + server + "/myemsl/cgi-bin/finish" + location
						r, err = client.Get(finalize)
						if err != nil {
							log.Printf("Failed to talk to server.")
							continue
						}
						log.Printf("finish: %v\n", r.StatusCode)
						if r.StatusCode != 200 {
							in_outage = true
							log.Printf("Got strange status code from finish. %v", int(r.StatusCode))
							break
						}
						readbuf = bufio.NewReader(r.Body)
						status := ""
						for {
							data, isPrefix, err := readbuf.ReadLine();
							if(isPrefix) {
								log.Println("Line too long");
								break
							} else if(err == io.EOF) {
								break;
							} else if(err != nil) {
								log.Println(err)
								break
							}
							str := string(data)
							if(strings.HasPrefix(str, "Status:")) {
								status = strings.TrimSpace(str[len("Status:"):])
							} else if(strings.HasPrefix(str, "Outage:")) {
								outage = strings.TrimSpace(str[len("Outage:"):])
							}
						}
						if outage != "" {
							in_outage = true
							break
						}
						if status == "" {
							log.Printf("Failed to finish. Status not retrieved and not in outage.\n")
							in_outage = true
							break
						}
						log.Printf("Uploaded. %v %v.\n", status)
//FIXME verify file service exists in cred. exists or isnt ""
						err = bm.bundleStringSet(user, id, "file_service", creds.Services["files"])
						if err != nil {
							log.Printf("Failed to set file service. %v\n", err)
							continue
						}
						err = bm.bundleStringSet(user, id, "status_url", status)
						if err != nil {
							log.Printf("Failed to set status_url. %v\n", err)
							continue
						}
						newstate := BundleState_Submitted
						err = bm.BundleStateSet(user, id, newstate)
						if err != nil {
							log.Printf("Failed to transition bundle from bundling to upload. %v\n", err)
							break
						}
						log.Printf("Put: %v\n",	 resp)
					}
					if in_outage {
						t := time.Now()
						self.cm.outage[user] = &t
						continue
					}
					if usercreds_bad {
						continue
					}
				}
			}
			if found == false {
				self.wake.L.Lock()
				users, err := self.bm.bundleUsersForState(BundleState_ToUpload, self.cm)
				if err == nil && len(users) < 1 {
					self.wake.Wait()
				}
				self.wake.L.Unlock()
			}
		}
	} ()
	return self
}

//Must not be called with sql lock held
func (self *uploader) Wakeup() {
	self.wake.L.Lock()
	self.wake.Signal()
	self.wake.L.Unlock()
}

func uploaderInit() {

}
