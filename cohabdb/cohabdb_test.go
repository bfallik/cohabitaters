package cohabdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
)

func Test_CreateOrSelectUser(t *testing.T) {
	ctx := context.Background()

	db, err := Open()
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer db.Close()

	if err := CreateTables(ctx, db); err != nil {
		t.Fatalf("%v", err)
	}
	queries := New(db)

	cup0 := CreateUserParams{
		Sub: "foo",
		Name: sql.NullString{
			String: "bar",
			Valid:  true,
		},
	}
	user, err := CreateOrSelectUser(ctx, queries, cup0)
	if err != nil {
		t.Errorf("%v", err)
	}

	if user.Sub != cup0.Sub {
		t.Errorf("sub got = %v, want = %v", user.Sub, cup0.Sub)
	}

	if !cmp.Equal(user.Name, cup0.Name) {
		t.Errorf("full name got = %v, want = %v", user.Name, cup0.Name)
	}

	cup0.Name = sql.NullString{
		String: "baz",
		Valid:  true,
	}

	user, err = CreateOrSelectUser(ctx, queries, cup0)
	if err != nil {
		t.Errorf("%v", err)
	}

	if user.Sub != cup0.Sub {
		t.Errorf("sub got = %v, want = %v", user.Sub, cup0.Sub)
	}

	if !cmp.Equal(user.Name, cup0.Name) {
		t.Errorf("full name got = %v, want = %v", user.Name, cup0.Name)
	}

}

func TestQueries(t *testing.T) {
	ctx := context.Background()

	db, err := Open()
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer db.Close()

	if err := CreateTables(ctx, db); err != nil {
		t.Errorf("%v", err)
	}
	queries := New(db)

	user, err := queries.CreateUser(ctx, CreateUserParams{Name: sql.NullString{String: "Test User", Valid: true}, Sub: "Test Sub"})
	if err != nil {
		t.Errorf("%v", err)
	}

	insertedTok := oauth2.Token{
		AccessToken: "hello",
	}
	insertedTokJSON, err := json.Marshal(insertedTok)
	if err != nil {
		t.Errorf("%v", err)
	}

	csp := CreateSessionParams{
		UserID: sql.NullInt64{Int64: user.ID, Valid: true},
	}
	session, err := queries.CreateSession(ctx, csp)
	if err != nil {
		t.Errorf("%v", err)
	}

	err = queries.UpdateTokenBySession(ctx, UpdateTokenBySessionParams{
		ID:    session.ID, // need to name this argument
		Token: sql.NullString{String: string(insertedTokJSON), Valid: true},
	})
	if err != nil {
		t.Errorf("%v", err)
	}

	record, err := queries.GetToken(ctx, session.ID)
	if err != nil {
		t.Errorf("%v", err)
	}

	var fetchedTok oauth2.Token
	if err := json.Unmarshal([]byte(record.String), &fetchedTok); err != nil {
		t.Errorf("%v", err)
	}

	if !cmp.Equal(fetchedTok, insertedTok, cmpopts.IgnoreFields(fetchedTok, "raw", "expiryDelta")) {
		t.Errorf("GetToken() = %+v, want %+v", fetchedTok, insertedTok)
	}
}
