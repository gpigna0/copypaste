CREATE TABLE IF NOT EXISTS users (
  id       UUID  UNIQUE NOT NULL,
  username VARCHAR(25) PRIMARY KEY,
  password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS clipboard (
  id        UUID PRIMARY KEY,
  clip_text TEXT NOT NULL,
  username  VARCHAR(25) NOT NULL,

  CONSTRAINT fk_users
    FOREIGN KEY (username) REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS files (
  id       UUID PRIMARY KEY,
  filename TEXT UNIQUE NOT NULL,
  username VARCHAR(25) NOT NULL,

  CONSTRAINT fk_users
    FOREIGN KEY (username) REFERENCES users(username)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);
