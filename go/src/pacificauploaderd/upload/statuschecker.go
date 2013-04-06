package upload

import (
	"io"
	"log"
	"sync"
	"time"
	"net/http"
	"encoding/json"
)

import (
	pacificaauth "pacifica/auth"
)

const (
	_STATUS_CHECKER_SLOW_CHECK_TIME = 5 * time.Minute
	_STATUS_CHECKER_QUICK_CHECK_TIME = 10 * time.Second
)

type statusChecker struct {
	wake *sync.Cond
	bm *BundleManager
	cm *CredsManager
	stopChan chan bool
}

type substatusStructure struct {
	State int
	Status string
	Message string
}

type statusStructure struct {
	MyEMSLStatus []substatusStructure
	Transaction string
}

func statusCheckerNew(bm *BundleManager, cm *CredsManager) *statusChecker {
	self := &statusChecker{bm: bm, cm: cm, stopChan: make(chan bool), wake: sync.NewCond(&sync.Mutex{})}
	err := bm.BundleStateChangeWatchSet(BundleState_Submitted, func(user string, bundle_id int, state BundleState) { self.Wakeup() })
	if err != nil {
		log.Printf("Failed to register wakeup callback with the bundle manager.")
		return nil
	}
	self.Run()
	return self
}

func (self *statusChecker) Run() {
	go func() {
		waitChan := make(chan bool)
		for {
			toQuickCheck := false
			ids, err := self.bm.bundleIdsForStateGet(BundleState_Submitted)
			if err == nil && ids != nil {
				for user, list := range ids {
					tmpToQuickCheck := self.statusUserCheck(user, list)
					if tmpToQuickCheck {
						toQuickCheck = tmpToQuickCheck
					}
				}
			}
			duration := _STATUS_CHECKER_SLOW_CHECK_TIME
			if toQuickCheck {
				duration = _STATUS_CHECKER_QUICK_CHECK_TIME
			}
			timer := time.NewTimer(duration)
			go self.wait(waitChan)
			select {
				case <- waitChan:
					timer.Stop()
				case <- self.stopChan:
					return
				case <- timer.C:
			}
		}
	} ()
}

func (self *statusChecker) Stop() {
	self.stopChan <- true
}

func (self *statusChecker) statusIdCheck(user string, creds *pacificaauth.Auth, client *http.Client, id int) (usercreds_bad bool, quickCheck bool) {
	status_url, err := self.bm.BundleStringGet(user, id, "status_url")
	if err != nil || status_url == "" {
		log.Printf("Failed to get status url from %v. %v\n", id, err)
//FIXME flag bundle for error if status_url == ""?
		return false, false
	}
	log.Printf("Got status to check.. %v %v %v\n", user, id, status_url)
//FIXME deal with outage page.
	res, err := client.Get(status_url + "/json")
	if err != nil {
		log.Printf("Failed to talk to server.")
		return false, false
	}
	defer res.Body.Close()
	log.Printf("status_url: %v\n", res.StatusCode)
	if res.StatusCode == 401 {
		log.Printf("Users creds are too old. Invalidating. %v\n", user)
		self.cm.userCreds[user] = nil
		return true, false
	}
	if res.StatusCode != 200 {
//FIXME
	}
	var status statusStructure
	dec := json.NewDecoder(res.Body)
	for {
		if err := dec.Decode(&status); err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Error %v\n", err)
			break;
		}
	}
	if err != nil {
//FIXME transient error. Update error message?
		return false, false
	}
	if len(status.MyEMSLStatus) < 7 {
		log.Printf("The correct number of elements where not returned in the status page.\n");
//FIXME transient error. Update error message?
		return false, false
	}
	log.Printf("Got %v states\n", len(status.MyEMSLStatus))
	success := 0
	error := false
	errormsg := ""
	quickCheck = true
	if status.Transaction != "" {
		log.Printf("Setting transaction to %v\n", status.Transaction)
//FIXME error handle.
		bm.bundleStringSet(user, id, "trans_id", status.Transaction)
	}
	for pos, item := range status.MyEMSLStatus {
		if item.State != pos {
			log.Printf("Badly formatted status file!\n")
//FIXME transient error. Update error message?
			return false, false
		}
		if item.Status == "ERROR" {
			log.Printf("Error at %v %s\n", status, status_url)
			error = true
			errormsg = item.Message
			quickCheck = false
		}
		if item.State == 5 && item.Status == "SUCCESS" {
			log.Printf("Available %v\n", true)
//FIXME error handle.
			bm.bundleBoolSet(user, id, "available", true)
			quickCheck = false
		}
		if item.Status == "SUCCESS" {
			success++
		}
	}
	var newstate BundleState
	if error {
		newstate = BundleState_Error
		log.Printf("Bundle failed with %v.", errormsg)
//FIXME error handle.
		bm.bundleStringSet(user, id, "error", errormsg)
	}
	if success == len(status.MyEMSLStatus) {
		newstate = BundleState_Safe
	}
	if error || success == len(status.MyEMSLStatus) {
		err = bm.BundleStateSet(user, id, newstate)
		if err != nil {
			log.Printf("Failed to update state from the status checker. %v\n", err)
		}
	}
	return false, quickCheck
}

func (self *statusChecker) statusUserCheck(user string, ids []int) (quickCheck bool) {
	usercreds_bad := false
	creds := self.cm.userCreds[user]
	if creds == nil {
		log.Printf("No creds for user %v\n", user)
		return
	}
	client := creds.NewClient()
	for _, id := range ids {
		var tmpQuickCheck bool
		usercreds_bad, tmpQuickCheck = self.statusIdCheck(user, creds, client, id)
		if tmpQuickCheck {
			quickCheck = tmpQuickCheck
		}
		if usercreds_bad {
			break
		}
	}
	return quickCheck
}

//Must not be called with sql lock held
func (self *statusChecker) Wakeup() {
	self.wake.L.Lock()
	self.wake.Signal()
	self.wake.L.Unlock()
}

func (self *statusChecker) wait(waitChan chan<- bool) {
	self.wake.L.Lock()
	self.wake.Wait()
	self.wake.L.Unlock()
	waitChan <- true
}

