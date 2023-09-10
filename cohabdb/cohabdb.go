package cohabdb

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"

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

func CreateOrSelectUser(ctx context.Context, queries *Queries, params CreateUserParams) (User, error) {
	u, err := queries.CreateUser(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// already exists
			return queries.GetUserBySub(ctx, params.Sub)
		}
		return User{}, err
	}
	return u, nil
}
