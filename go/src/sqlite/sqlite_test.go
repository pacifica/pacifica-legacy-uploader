package sqlite

import (
	"testing"
)

// Test that the version number is returned correctly
func TestVersion(t *testing.T) {
	var v = Version()

	if len(v) == 0 {
		t.Error("Nil version or version of 0 length returned")
	}
}
