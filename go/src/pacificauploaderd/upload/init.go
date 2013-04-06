package upload

import "log"

/*
	Default BundleManager.  Set during upload.Init and does
	not change thereafter.
*/
var bm *BundleManager

func Init() *BundleManager {
	log.Println("Upload subsystem init.")
	cm := credsInit()
	bm := bundleManagerInit()
	if bm == nil {
		return nil
	}
	bundlerInit()
	bdlr := bundlerNew(bm)
	if bdlr == nil {
		return nil
	}
	uploaderInit()
	uplr := uploaderNew(bm, cm)
	if uplr == nil {
		return nil
	}
	sc := statusCheckerNew(bm, cm)
	if sc == nil {
		return nil
	}
	return bm
}
