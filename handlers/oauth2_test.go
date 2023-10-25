package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bfallik/cohabitaters/cohabdb"
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

func Test_LogUserIn(t *testing.T) {
	ctx := context.Background()

	const origSessionID = 7
	origUser := cohabdb.InsertUserParams{Sub: "foobar"}

	tests := []struct {
		Desc    string
		SetupFn func(q *cohabdb.Queries) error
	}{
		{
			Desc:    "user does not exist",
			SetupFn: func(q *cohabdb.Queries) error { return nil },
		},
		{
			Desc: "user exists",
			SetupFn: func(q *cohabdb.Queries) error {
				_, err := q.InsertUser(ctx, origUser)
				return err
			},
		},
		{
			Desc: "user and session exist",
			SetupFn: func(q *cohabdb.Queries) error {
				user, err := q.InsertUser(ctx, origUser)
				if err != nil {
					return err
				}

				if _, err = q.InsertSession(ctx, cohabdb.InsertSessionParams{
					ID:     int64(origSessionID),
					UserID: user.ID,
				}); err != nil {
					return err
				}

				return q.ExpireSession(ctx, int64(origSessionID))
			},
		},
	}

	/*

	 */

	for _, test := range tests {
		t.Run(test.Desc, func(t *testing.T) {
			db, err := cohabdb.OpenInMemory()
			if err != nil {
				t.Fatalf("%v", err)
			}
			defer db.Close()

			if err := cohabdb.CreateTables(ctx, db); err != nil {
				t.Errorf("%v", err)
			}
			queries := cohabdb.New(db)
			o := Oauth2{Queries: queries}

			if err := test.SetupFn(queries); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sess, err := o.LogUserIn(ctx, cohabdb.UpsertUserParams{Sub: origUser.Sub}, origSessionID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !cmp.Equal(sess.ID, int64(origSessionID)) {
				t.Errorf("unexpected session ID, got: %v, want: %v", sess.ID, origSessionID)
			}

			if !sess.IsLoggedIn {
				t.Errorf("unexpected user not logged in")
			}
		})
	}

	t.Run("existing session, new user", func(t *testing.T) {
		db, err := cohabdb.OpenInMemory()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer db.Close()

		if err := cohabdb.CreateTables(ctx, db); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		queries := cohabdb.New(db)
		o := Oauth2{Queries: queries}

		user, err := queries.InsertUser(ctx, origUser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		sess, err := queries.InsertSession(ctx, cohabdb.InsertSessionParams{
			ID:     int64(origSessionID),
			UserID: user.ID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := o.LogUserIn(ctx, cohabdb.UpsertUserParams{Sub: "bazboo"}, int(sess.ID)); err == nil {
			t.Errorf("missing expected error")
		}
	})
}
