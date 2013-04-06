/*
	Implements an automatic file disocovery, filtering, and notification component.
*/
package auto

import (
	"log"
	"pacificauploaderd/upload"
)

/*
	Default watcher
*/
var w *watcher

/*
	Default matcher
*/
var m *matcher

/* 
	Default fileStateManager
*/
var fsm *fileStateManager

/*
	Default autoManager
*/
var am *autoManager

/*
	Default bundleManager provided by callee of auto.Init.  Not managed by
	auto package.
*/
var autoBm *upload.BundleManager

// Single function call to start up auto component
func Init(bm *upload.BundleManager) {
	log.Println("Auto subsystem init.")
	fileStateInit()
	watcherInit()
	matcherInit()
	configInit()
	autoInit(bm)
}
