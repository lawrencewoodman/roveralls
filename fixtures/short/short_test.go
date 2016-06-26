package short

import (
	"testing"
)

func TestAmIShort(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	} else {
		if !AmIShort() {
			t.Errorf("AmIShort got: false, want: true")
		}
	}
}
