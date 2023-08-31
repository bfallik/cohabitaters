CREATE TABLE authors (
  id   INTEGER PRIMARY KEY,
  name text    NOT NULL,
  bio  text
);

CREATE TABLE oauth2_tokens (
  id INTEGER PRIMARY KEY,
	access_token TEXT NOT NULL,
	token_type TEXT NOT NULL,
	refresh_token TEXT NOT NULL,
	expiry TIMESTAMP NOT NULL
)
