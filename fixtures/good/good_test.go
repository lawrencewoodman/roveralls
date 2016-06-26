package good

import (
	"testing"
)

func TestAmIGood(t *testing.T) {
	if !AmIGood() {
		t.Error("AmIGood() got: false, want: true")
	}
}
