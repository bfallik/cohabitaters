CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  sub TEXT UNIQUE NOT NULL,
  full_name TEXT,
  token TEXT
);

CREATE TABLE sessions (
  id INTEGER PRIMARY KEY,
  user_id INTEGER,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
  is_logged_in BOOLEAN NOT NULL DEFAULT (true),
  FOREIGN KEY(user_id) REFERENCES users(id)
);
