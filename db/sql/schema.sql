CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  full_name TEXT NOT NULL
);

CREATE TABLE sessions (
  id INTEGER PRIMARY KEY,
  user_id INTEGER,
  expiry DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE tokens (
  id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL,
	token TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
