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

	m.Delete(7)
	got = m.Get(7)
	//lint:ignore ST1017 want the condition order to match the format string
	if 0 != got {
		t.Errorf("expected:11, got: %v", got)
	}

	callCount := 0
	iter := func(id int, val int) bool {
		callCount++
		return true
	}
	m.Range(iter)
	//lint:ignore ST1017 want the condition order to match the format string
	if 1 != callCount {
		t.Errorf("expected:0, got: %v", callCount)
	}

	m.Set(7, 11)
	m.Set(8, 11)
	m.Set(9, 11)
	m.Range(iter)
	//lint:ignore ST1017 want the condition order to match the format string
	if 4 != callCount {
		t.Errorf("expected:3, got: %v", callCount)
	}
}
