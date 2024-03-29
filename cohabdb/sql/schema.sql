CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  sub TEXT UNIQUE NOT NULL,
  name TEXT,
  picture TEXT,
  token TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
  id INTEGER UNIQUE NOT NULL,
  user_id INTEGER NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
  is_logged_in BOOLEAN NOT NULL DEFAULT (true),
	google_force_approval BOOLEAN NOT NULL DEFAULT (false),
	contact_groups_json TEXT,
	selected_resource_name TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id),
  CONSTRAINT unique_user_id UNIQUE(id, user_id)
);
