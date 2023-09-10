package handlers

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_mapGet(t *testing.T) {
	m := map[string]interface{}{"foo": "bar"}

	t.Run("exists", func(t *testing.T) {
		val, ok := mapGet[string](m, "foo")
		if !ok {
			t.Errorf("key foo not found")
		}
		if val != "bar" {
			t.Errorf("incorrect value, got %v, want %v", val, m["foo"])
		}
	})
	t.Run("does not exist", func(t *testing.T) {
		_, ok := mapGet[string](m, "baz")
		if ok {
			t.Errorf("unexpected key baz found")
		}
	})
	t.Run("wrong type", func(t *testing.T) {
		_, ok := mapGet[int](m, "foo")
		if ok {
			t.Errorf("unexpected int foo found")
		}
	})
}

var nullStringComparer = cmp.Comparer(func(got sql.NullString, want sql.NullString) bool {
	switch {
	case !got.Valid && !want.Valid:
		return true
	case got.Valid && want.Valid:
		return got.String == want.String
	default:
		return false
	}
})

func Test_claimToNullString(t *testing.T) {
	m := map[string]interface{}{"foo": "bar"}

	t.Run("exists", func(t *testing.T) {
		got := claimToNullString(m, "foo")
		if !cmp.Equal(got, sql.NullString{String: "bar", Valid: true}) {
			t.Errorf("unexpected NullString, got: %v", got)
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		got := claimToNullString(m, "bar")
		if !cmp.Equal(got, sql.NullString{}, nullStringComparer) {
			t.Errorf("unexpected NullString, got: %v", got)
		}
	})
}
