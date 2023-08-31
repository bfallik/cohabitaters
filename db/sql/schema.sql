CREATE TABLE authors (
  id   INTEGER PRIMARY KEY,
  name text    NOT NULL,
  bio  text
);

CREATE TABLE oauth2_tokens (
  id INTEGER PRIMARY KEY,
	token TEXT NOT NULL
)
