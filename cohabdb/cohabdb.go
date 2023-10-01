package cohabdb

import (
	"context"
	"database/sql"
	_ "embed"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql/schema.sql
var ddl string

func Open(filename string) (*sql.DB, error) {
	u := url.URL{}
	u.Path = filename
	q := u.Query()
	q.Set("_foreign_keys", "on")
	u.RawQuery = q.Encode()
	return sql.Open("sqlite3", u.RequestURI())
}

func CreateTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ddl)
	return err
}
