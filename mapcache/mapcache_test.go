package mapcache

import (
	"testing"
)

func TestMap(t *testing.T) {
	m := Map[int]{}
	got := m.Get(7)
	//lint:ignore ST1017 want the condition order to match the format string
	if 0 != got {
		t.Errorf("expected: 0, got: %v", got)
	}

	m.Set(7, 11)
	got = m.Get(7)
	//lint:ignore ST1017 want the condition order to match the format string
	if 11 != got {
		t.Errorf("expected:11, got: %v", got)
	}
}
