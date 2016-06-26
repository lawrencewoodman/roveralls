package good2

import (
	"testing"
)

func TestAmIGood2(t *testing.T) {
	if !AmIGood2() {
		t.Error("AmIGood2() got: false, want: true")
	}
}
