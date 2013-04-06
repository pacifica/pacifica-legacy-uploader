package upload

type BundleState int

const (
	BundleState_Unsubmitted BundleState = 1
	BundleState_ToBundle BundleState = 2
	BundleState_ToUpload BundleState = 3
	BundleState_Submitted BundleState = 4
	BundleState_Error BundleState = 5
	BundleState_Safe BundleState = 6
)

type BundleAction int

const (
	BundleAction_Editable BundleAction = iota
	BundleAction_Submittable
	BundleAction_MakeEditable
	BundleAction_Deletable
	BundleAction_Cancelable
)

func BundleStateTransitionOk(old BundleState, new BundleState) bool {
	if old == BundleState_Unsubmitted && new == BundleState_ToBundle {return true}
	if old == BundleState_ToBundle && (new == BundleState_ToUpload || new == BundleState_Error) {return true}
	if old == BundleState_ToUpload && (new == BundleState_Submitted || new == BundleState_Error) {return true}
	if old == BundleState_Submitted && (new == BundleState_Safe || new == BundleState_Error) {return true}
	if old == BundleState_Error && new == BundleState_Unsubmitted {return true}
	return false
}	 

func BundleStateActionOk(state BundleState, action BundleAction) bool {
	if state == BundleState_Unsubmitted && (action == BundleAction_Editable || action == BundleAction_Deletable || action == BundleAction_Submittable) {return true}
	if state == BundleState_ToBundle && action == BundleAction_Cancelable {return true}
	if state == BundleState_ToUpload && action == BundleAction_Cancelable {return true}
	if state == BundleState_Error && (action == BundleAction_Deletable || action == BundleAction_MakeEditable) {return true}
	if state == BundleState_Safe && (action == BundleAction_Deletable || action == BundleAction_MakeEditable) {return true}
	return false
}
