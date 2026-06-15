package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err != nil {
		return nil, err
	}
	if _, err = db.Exec(schema); err != nil {
		return nil, err
	}
	if err = migratePreciousMetalType(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migratePreciousMetalType(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(precious_metals)`)
	if err != nil {
		return err
	}
	hasName, hasType := false, false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		hasName = hasName || name == "name"
		hasType = hasType || name == "type"
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if hasName && !hasType {
		_, err = db.Exec(`ALTER TABLE precious_metals RENAME COLUMN name TO type`)
	}
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 username TEXT NOT NULL UNIQUE COLLATE NOCASE,
 password_hash TEXT NOT NULL,
 full_name TEXT NOT NULL,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS refresh_tokens (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_hash TEXT NOT NULL UNIQUE,
 expires_at DATETIME NOT NULL,
 revoked_at DATETIME,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS precious_metals (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 type TEXT NOT NULL,
 brand TEXT NOT NULL,
 bought_price REAL NOT NULL CHECK (bought_price >= 0),
 buy_price REAL NOT NULL DEFAULT 0 CHECK (buy_price >= 0),
 buyback_price REAL NOT NULL DEFAULT 0 CHECK (buyback_price >= 0),
 weight REAL NOT NULL CHECK (weight > 0),
 price_updated_at DATETIME,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_metals_user_id ON precious_metals(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_user_id ON refresh_tokens(user_id);
`
