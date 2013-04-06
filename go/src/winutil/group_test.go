package winutil

import (
	"fmt"
	"testing"
)

//Test IsnInGroup.  Note, you will need to change the usernames that are
//being tested depending on your machine.
func TestIsInGroup(t *testing.T) {
	fmt.Printf("Entering TestIsInGroup\n")
	defer fmt.Printf("Leaving TestIsInGroup\n")

	user := "WE19886\\Renamed_Admin"
	group := "Administrators"
	result, _ := IsInLocalGroup(user, group)
	fmt.Printf("%v in %v == %v\n", user, group, result)

	user = "PNL\\d3m306"
	group = "Administrators"
	result, _ = IsInLocalGroup(user, group)
	fmt.Printf("%v in %v == %v\n", user, group, result)

	user = "WE19886\\Bubba"
	group = "Administrators"
	result, _ = IsInLocalGroup(user, group)
	fmt.Printf("%v in %v == %v\n", user, group, result)
}
