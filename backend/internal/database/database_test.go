package database

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestOpenMigratesNameColumnToType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE precious_metals (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		brand TEXT NOT NULL,
		bought_price REAL NOT NULL,
		buy_price REAL NOT NULL,
		buyback_price REAL NOT NULL,
		weight REAL NOT NULL,
		price_updated_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	); INSERT INTO precious_metals(user_id,name,brand,bought_price,buy_price,buyback_price,weight) VALUES(1,'Gold','ANTAM',1,2,3,4)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	migrated, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer migrated.Close()
	var metalType string
	if err := migrated.QueryRow(`SELECT type FROM precious_metals WHERE id=1`).Scan(&metalType); err != nil {
		t.Fatal(err)
	}
	if metalType != "Gold" {
		t.Fatalf("expected Gold, got %q", metalType)
	}
}
