package db

import (
	"context"
	"database/sql"
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
	_, err = queries.CreateOauth2Token(ctx, cohabdb.CreateOauth2TokenParams{
		ID:           1,
		AccessToken:  insertedTok.AccessToken,
		TokenType:    insertedTok.TokenType,
		RefreshToken: insertedTok.RefreshToken,
		Expiry:       insertedTok.Expiry,
	})
	if err != nil {
		t.Errorf("%v", err)
	}

	rawTok, err := queries.GetOauth2Token(ctx, 1)
	if err != nil {
		t.Errorf("%v", err)
	}
	fetchedTok := oauth2.Token{
		AccessToken:  rawTok.AccessToken,
		TokenType:    rawTok.TokenType,
		RefreshToken: rawTok.RefreshToken,
		Expiry:       rawTok.Expiry,
	}

	if insertedTok.AccessToken != fetchedTok.AccessToken {
		t.Errorf("mismatched tokens")
	}
}
