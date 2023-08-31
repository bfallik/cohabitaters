package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/bfallik/cohabitaters/db/cohabdb"
	"golang.org/x/oauth2"
)

func TestQueries(t *testing.T) {
	ctx := context.Background()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("%v", err)
	}

	if err := CreateTables(ctx, db); err != nil {
		t.Errorf("%v", err)
	}
	queries := cohabdb.New(db)

	insertedTok := oauth2.Token{
		AccessToken: "hello",
	}
	insertedTokJSON, err := json.Marshal(insertedTok)
	if err != nil {
		t.Errorf("%v", err)
	}

	_, err = queries.CreateOauth2Token(ctx, cohabdb.CreateOauth2TokenParams{
		ID:    1,
		Token: string(insertedTokJSON),
	})
	if err != nil {
		t.Errorf("%v", err)
	}

	record, err := queries.GetOauth2Token(ctx, 1)
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
