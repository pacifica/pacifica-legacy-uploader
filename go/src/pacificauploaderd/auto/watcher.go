package auto

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"pacificauploaderd/common"
)

const (
	_NANO_SECONDS_IN_DAY         = 1e9 * 86400
	_STATE_RETENTION_TIME        = _NANO_SECONDS_IN_DAY * 60 //60 days
	_DEFAULT_WATCH_DELAY_SECONDS = 10
)

type foundFunc func(searchPath, foundPath string)
type walkedFunc func()

/*
	Finds files in paths and creates FileState from them.  Sends through found channel.
*/
type watcher struct {
	delay     time.Duration
	paths     []string     //List of paths this watcher should look for files in.  Can be directories or files.
	running   int32        //Flag for determining if the watcher is in use.	
	found     []foundFunc  //Called asynchronously when watcher encounters a file in one of its configured paths.	
	walked    []walkedFunc //Called asynchronously when watcher has completed an iteration of all configured paths.
	pathMutex sync.Mutex
}

func newWatcher(runDelaySeconds int64) *watcher {
	tmp := new(watcher)
	tmp.delay = time.Duration(runDelaySeconds * int64(time.Second))
	tmp.paths = []string{}
	tmp.found = []foundFunc{}
	tmp.walked = []walkedFunc{}
	return tmp
}

func (self *watcher) updatePaths(newPaths []string) {
	self.pathMutex.Lock()
	defer self.pathMutex.Unlock()
	self.paths = newPaths
}

func (self *watcher) watch() error {
	//Reentrant calls will just exit...
	if !atomic.CompareAndSwapInt32(&self.running, 0, 1) {
		return nil
	}
	defer func() {
		if !atomic.CompareAndSwapInt32(&self.running, 1, 0) {
			log.Panic("Watch running and could not swap state")
		}
	}()

	//Async loop...
	go func() {
		for {
			self.processPaths()
			self.callWalked()
			t := time.NewTimer(self.delay)
			<-t.C
		}
	}()
	return nil
}

func (self *watcher) processPaths() {
	self.pathMutex.Lock()
	tmpPaths := self.paths
	self.pathMutex.Unlock()
	for _, path := range tmpPaths {
		pathFi, err := os.Stat(path)
		if err != nil {
			common.Dprintf("Could not stat %s, %v", path, err)
			continue
		}

		if !pathFi.IsDir() {
			self.callFound(path, path)
		} else {
			w := func(newPath string, info os.FileInfo, err error) error {
				//FIXME work around broken Go!
				//See issue http://code.google.com/p/go/issues/detail?id=3486
				if info.IsDir() && err == nil {
					f, err := os.Open(newPath)
					if err != nil {
						return filepath.SkipDir
					}
					f.Close()
				}
				if info.IsDir() && err != nil {
					common.Dprintf("%s skipped, error %v\n", newPath, err)
					return filepath.SkipDir
				} else if !info.IsDir() {
					self.callFound(path, newPath)
				}
				return nil
			}
			err = filepath.Walk(path, w)
			if err != nil {
				common.Dprintf("Walk failed %v with %v\n", path, err)
			}
		}
	}
}

//FIXME This looks like it will spawn too many in some cases. Rate limit this somehow?
//NGT comment 4/24/2012, not sure we need to worry about this.  Goroutines should be able to spawned without
//taking a major hit. One goroutine does not equal one OS thread...With Go1, all goroutines run on a single
//thread using cooperative sheduling, but that can be tweaked using GOMAXPROCS, and we should see that
//evolve over time...many people write programs with hundreds or thousands of goroutines without issue.
//KMF comment 4/24//2013. This indeed is spawning threads faster then can be processed leading to memory leaks
//and crashes.
func (self *watcher) callFound(searchPath, foundPath string) {
	if self.found == nil {
		return
	}
	for _, f := range self.found {
		//go f(searchPath, foundPath)
		f(searchPath, foundPath)
	}
}

//FIXME callWalked gets called before all processing threads are done.
//NGT comment 4/24/2012 2 possible solutions
//1. Change callFound to not spawn goroutines, especially if it isn't providing us anything useful.
//
//2. type foundFunc func(searchPath, foundPath string) includes a channel it writes to on completion.
//   callWalked selects on each element in self.found
func (self *watcher) callWalked() {
	if self.walked == nil {
		return
	}
	for _, f := range self.walked {
		//go f()
		f()
	}
}

func watcherInit() {
	w = newWatcher(_DEFAULT_WATCH_DELAY_SECONDS)
	w.watch()
}
