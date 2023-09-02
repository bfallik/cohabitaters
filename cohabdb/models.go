// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0

package cohabdb

import (
	"database/sql"
	"time"
)

type Session struct {
	ID     int64
	UserID sql.NullInt64
	Expiry time.Time
}

type Token struct {
	ID     int64
	UserID int64
	Token  string
}

type User struct {
	ID       int64
	FullName string
}