package auto

import (
	"errors"
	"log"
	"pacificauploaderd/common"
	"pacificauploaderd/upload"
	"sync"
	"time"
)

var (
	autoTimerFreq      time.Duration = time.Second * 10
	bundleModThreshold time.Duration = time.Minute * 1
	maxNumFilesBundle  int64         = 10e3 //10K files
)

type autoFile struct {
	fs          *fileState
	newFileName string
	groups      [][2]string
}

type autoManager struct {
	workingBundles map[string]*userBundles
	//If a bundle is submitted, deleted, or met thresholds (e.g. max # of files), this flag will be set to true and 
	//stateDatabase.getWorkingBundles will be called when the working bundles are needed again.
	workingBundlesNeedsRefresh bool
	stateManager               *fileStateManager
	mutex                      sync.Mutex
}

func newAutoManager(stateManager *fileStateManager) *autoManager {
	self := new(autoManager)
	self.stateManager = stateManager
	self.refreshWorkingBundlesNoLock(true)
	self.recover()
	return self
}

// Use for sending a file to be managed by the auto manager.  The auto manager will decide if and when the file should
// be sent to the bundle manager for upload.
func (self *autoManager) addAutoFile(fs fileState, newFileName string, groups [][2]string) {
	common.Dprintln("Entering addAutoFile")
	defer common.Dprintln("Leaving addAutoFile")

	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.refreshWorkingBundlesNoLock(false)

	common.Dprintf("addAutoFile working with %+v", fs)

	//Skip this file if it has been seen before.
	if fs.passOff != notSeenBefore {
		common.Dprintf("file %s, %s, %s, has been seen before...skipping.", fs.userName, fs.ruleName, fs.fullPath)
		return
	}

	af := &autoFile{fs: &fs, newFileName: newFileName, groups: groups}

	//Get the corresponding WatchRule for this fileState.
	wr := getWatchRule(fs.userName, fs.ruleName)
	if wr == nil {
		log.Printf("getWatchRule returned no watchRule for user %s and rule %s, addAutoFile will be re-attempted later.",
			fs.userName, fs.ruleName)
		return
	}

	//Bundle the file and log any error.
	err := self.bundleAutoFileNoLock(af, fs.userName, fs.ruleName, wr)
	if err != nil {
		log.Printf("bundleAutoFileNoLock failed with error %v, addAutoFile will be re-attempted later.", err)
		return
	}
}

func (self *autoManager) bundleAutoFileNoLock(af *autoFile, userName string, ruleName string, wr *WatchRule) error {
	//Check arguments.
	if af == nil {
		return errors.New("af must not be nil")
	}
	if userName == "" {
		return errors.New("user must not be empty string")
	}
	if wr == nil {
		return errors.New("wr must not be nil")
	}

	//Get the userBundles for this user, if there are any, otherwise create them.
	bundles, ok := self.workingBundles[userName]
	if !ok {
		bundles = &userBundles{user: userName}
	}

	//Get or create the bundle id and BundleMD we will be working with.
	var bid *int
	isAuto := wr.AutoSubmit
	isNewBundle := false
	if isAuto {
		bid = bundles.autoSubmitBid
	} else {
		bid = bundles.noAutoSubmitBid
	}

	var b *upload.BundleMD
	var err error
	if bid != nil {
		b, err = autoBm.BundleGet(userName, *bid)
		if err == upload.ErrBundleNotFound {
			bid = nil
		} else if err != nil {
			return err
		} else {
			state, err := b.StateGet()
			if err != nil {
				return err
			}
			if state != upload.BundleState_Unsubmitted {
				bid = nil
			} else {
//FIXME is creating a temp var, getting its address and leaving the code block really safe?
				tmp := b.IdGet()
				bid = &tmp

				count, err := b.BundleFileCountGet()
				if err != nil {
					return err
				}

				if count >= maxNumFilesBundle && isAuto {
					b.Submit()
					bid = nil
				}
			}
		}
	}
	
	if bid == nil {
		isNewBundle = true
		//No good bundle already exists for this user. Create one.
		b, err = autoBm.BundleAdd(userName)
		if err != nil {
			return err
		}

		tmp := b.IdGet()
		bid = &tmp
		b.WatchAdd("auto")
	}
	
	//Add this autoFile to the bundle.
	bf, err := b.FileAdd(af.newFileName, af.fs.fullPath, false)
	if err != nil {
		return err
	}
	
	//Add groups
	err = bf.GroupsSet(af.groups)
	if err != nil {
		return err
	}	
	
	//Set Disable on Error
	err = bf.DisableOnErrorSet(true)
	if err != nil {
		return err
	}

	if isNewBundle {
		//Save the changes to bunndles
		if isAuto {
			bundles.autoSubmitBid = bid
		} else {
			bundles.noAutoSubmitBid = bid
		}
		self.stateManager.db.setUserBundles(bundles)
		//Our working bundles list is now out of date.  Set to true to trigger a query to
		//the stateDatabase on the next pass.
		self.workingBundlesNeedsRefresh = true
	}

	tmp := bf.IdGet()
	bfid := &tmp

	af.fs.passOff = addingToBundle
	af.fs.bundleId = nil
	af.fs.bundleFileId = nil
	err = self.stateManager.setFileState(af.fs)
	if err != nil {
		return err
	}
	
	//Commit the FileAdd started above
	err = bf.Commit()

	//There was an error committing the add above.  The file state manager
	//should retry the file at a later time.
	if err != nil {
		af.fs.passOff = notSeenBefore
		af.fs.bundleId = nil
		af.fs.bundleFileId = nil
		err = self.stateManager.setFileState(af.fs)
		if err != nil {
			//In this case, both the FileAdd commit and the setFileState have failed.
			//Something is really wrong if this code executes.
			log.Fatalf("Cannot set the state for %+v in bundleAutoFileNoLock.", err)
		}
		return err
	}
	
	//Set the fileState to inBundle and the bundleId and bundleFileId
	af.fs.passOff = inBundle
	af.fs.bundleId = bid
	af.fs.bundleFileId = bfid
	err = self.stateManager.setFileState(af.fs)
	if err != nil {
		//In this case, FileAdd, commit, and setFileState have worked, but then the last setFileState failed.
		//Something is really wrong if this code executes.
		log.Fatalf("Cannot set the state for %+v in bundleAutoFileNoLock.", err)
	}
	
	return nil
}

func (self *autoManager) refreshWorkingBundlesNoLock(force bool) {
	if force || self.workingBundlesNeedsRefresh {
		common.Dprintf("refreshing workingBundles")
		self.workingBundles = self.stateManager.db.getWorkingBundles()
		self.workingBundlesNeedsRefresh = false
	}
}

func (self *autoManager) runAuto() {
	go func() {
		for {
			common.Dprintf("runAuto sleeping for %v", autoTimerFreq)
			t := time.NewTimer(autoTimerFreq)
			<-t.C
			self.scheduleSubmit()
			self.monitorSubmitted()
		}
	}()
}

func (self *autoManager) scheduleSubmit() {
	common.Dprintln("Entering scheduleSubmit")
	defer common.Dprintln("Leaving scheduleSubmit")

	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.refreshWorkingBundlesNoLock(false)

	for _, v := range self.workingBundles {
		if v.autoSubmitBid == nil {
			continue
		}

		common.Dprintf("scheduleSubmit workingBundle %+v", *v)

		//Get the BundleMD for autoSubmitBid
		b, err := autoBm.BundleGet(v.user, *v.autoSubmitBid)
		if err != nil {
			log.Printf("BundleGet(%s, %d) failed %v", v.user, *v.autoSubmitBid, err)
			continue
		}

		//Get the amount of time since the last modification to the BundleMD
		modTime := time.Now().UnixNano() - v.autoSubmitLastTouched

		//Get the number of files in this BundleMD
		count, err := b.BundleFileCountGet()
		if err != nil {
			log.Printf("Failed to get file count for bundle id %d, error %v", v.autoSubmitBid, err)
			continue
		}

		common.Dprintf("modTime == %d, bundleModThreshold == %d", modTime, bundleModThreshold)

		//Submit only the autoSubmitBid if its activity has slowed or it has met the maximum number
		//of files threshold.
		if time.Duration(modTime) >= bundleModThreshold ||
			count >= maxNumFilesBundle {
			//Submit...
			err := b.Submit()
			if err != nil {
				log.Printf("Failed to submit bundle, error %v", err)
				continue
			}

			common.Dprintf("Bundle %d submitted!!!", v.autoSubmitBid)

			//Set the userBundles auto bundle to nil so a new one will get created on the next file add.
			self.workingBundlesNeedsRefresh = true
			v.autoSubmitBid = nil
			err = self.stateManager.db.setUserBundles(v)
			if err != nil {
				log.Printf("Failed to set userBundles %+v for, error %v", v, err)
				continue
			}
		}
	}
}

func (self *autoManager) monitorSubmitted() {
	common.Dprintln("Entering cleanupSubmitted")
	defer common.Dprintln("Leaving cleanupSubmitted")

	self.mutex.Lock()
	defer self.mutex.Unlock()

	users, bundle_ids, err := self.stateManager.db.getFileStatesProgressing()
	if err != nil {
		log.Printf("Failed to getFileStatesProgressing. error %v", err)
		return
	}
	var i int
	for i = 0; i < len(users); i++ {
		b, err := autoBm.BundleGet(users[i], bundle_ids[i])
		if err != nil {
			log.Printf("BundleGet(%s, %d) failed %v", users[i], bundle_ids[i], err)
			continue
		}
		bState, err := b.StateGet()
		if err != nil {
			log.Printf("Could not get state for BundleMD %+v, error %v", b, err)
			continue
		}

		if bState == upload.BundleState_Unsubmitted || bState == upload.BundleState_ToBundle || BundleState_Submitted {
			continue
		}

		bfs, err := b.FilesGet()
		if err != nil {
			log.Printf("FileIdsGet() returned error %v", err)
			continue
		}

		flagged, err := b.FlaggedToDelete();
		if err != nil {
			log.Printf("Failed to get flagged to delete returned error %v", err)
			continue
		}

		for _, bf := range bfs {
			bfStateMsg, err := bf.DisableOnErrorMsgGet()
			if err != nil {
				log.Printf("Could not get state for BundleFileMD %+v, error %v", bf, err)
				continue
			}

			//Get fileState(s) for this BundleFileMD
			bfid := bf.IdGet()
			states, err := self.stateManager.db.getFileStatesByBundleFileId(bfid)
			if err != nil || len(states) < 1 {
				log.Printf("Could not get any fileStates for bundle file id %d, error %v", bfid, err)
				continue
			}

			if bfStateMsg != "" ||
				bState == upload.BundleState_Error || (flagged && bState == upload.BundleState_Unsubmitted) {
				for _, v := range states {
					v.bundleId = nil
					v.bundleFileId = nil
					v.passOff = notSeenBefore
					self.stateManager.setFileState(v)
				}
				continue
			}

			if bState == upload.BundleState_Safe {
				for _, v := range states {
					v.passOff = done
					t, err := bf.MtimeGet()
					if err != nil {
						log.Printf("Could not get time for %+v, error %v", bf, err)
					} else {
						v.lastModified = t.UnixNano()
					}
					self.stateManager.setFileState(v)
				}
				continue
			}
		}

		if flagged ||
			bState == upload.BundleState_Safe ||
			bState == upload.BundleState_Error {
//FIXME consider this information with the above error handling. It probably should not delete state until all above is cleared.
			err := b.Delete("auto")
			if err != nil {
				log.Printf("Failed to delete BundleMD, error %v", err)
			}
		}
	}
}

func (self *autoManager) recover() {
	common.Dprintln("Entering recover")
	defer common.Dprintln("Leaving recover")

	self.mutex.Lock()
	defer self.mutex.Unlock()

	//Get list of fileStates in bundleManager that have autoPassOffState addingToBundle.
	states, err := self.stateManager.getFileStatesInLimbo()
	if err != nil {
		log.Fatalf("Could not get file states with state addingToBundle.  Error, %v", err)
	}

	for _, v := range states {
		log.Printf("Recovering file %s", v.fullPath)

		var isInBundle bool
		if v.bundleId != nil && v.bundleFileId != nil {
			b, err := autoBm.BundleGet(v.userName, *v.bundleId)
			if err != nil {
				isInBundle = false
			} else {
				_, err := autoBm.BundleFileGet(b, v.userName, *v.bundleId, *v.bundleFileId)
				if err != nil {
					isInBundle = false
				} else {
					isInBundle = true
				}
			}
		} else {
			isInBundle = false
		}

		if isInBundle {
			v.passOff = inBundle
		} else {
			v.bundleId = nil
			v.bundleFileId = nil
			v.passOff = notSeenBefore
		}

		err = self.stateManager.setFileState(v)
		if err != nil {
			log.Printf("Failed to set state on fileState %+v, error %v", v, err)
			continue
		}
	}
}

func autoInit(bm *upload.BundleManager) {
	autoBm = bm
	am = newAutoManager(fsm)
	am.runAuto()
}
