package cohabdb

import (
	"context"
	"database/sql"
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql/schema.sql
var ddl string

func Open() (*sql.DB, error) {
	return sql.Open("sqlite3", ":memory:?_foreign_keys=on")
}

func CreateTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, ddl)
	return err
}
