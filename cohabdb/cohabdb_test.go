package cohabdb

import (
	"context"
	"encoding/json"
	"testing"

	"golang.org/x/oauth2"
)

func TestQueries(t *testing.T) {
	ctx := context.Background()

	db, err := Open()
	if err != nil {
		t.Fatalf("%v", err)
	}

	if err := CreateTables(ctx, db); err != nil {
		t.Errorf("%v", err)
	}
	queries := New(db)

	user, err := queries.CreateUser(ctx, "Test User")
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

	_, err = queries.CreateToken(ctx, CreateTokenParams{
		ID:     1,
		UserID: user.ID,
		Token:  string(insertedTokJSON),
	})
	if err != nil {
		t.Errorf("%v", err)
	}

	record, err := queries.GetToken(ctx, 1)
	if err != nil {
		t.Errorf("%v", err)
	}

	var fetchedTok oauth2.Token
	if err := json.Unmarshal([]byte(record.Token), &fetchedTok); err != nil {
		t.Errorf("%v", err)
	}

	if insertedTok.AccessToken != fetchedTok.AccessToken {
		t.Errorf("mismatched tokens")
	}
}
