package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDBAndQueryAll(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer CloseDB()

	if _, err := DB().Exec(`
		CREATE TABLE cities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if _, err := DB().Exec(`INSERT INTO cities (name) VALUES ('Shanghai'), ('Hangzhou')`); err != nil {
		t.Fatalf("insert rows: %v", err)
	}

	type city struct {
		ID   int
		Name string
	}

	var got []city
	if err := QueryAll("SELECT id, name FROM cities ORDER BY id", &got, func(row Scanner) city {
		var c city
		if err := row.Scan(&c.ID, &c.Name); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		return c
	}); err != nil {
		t.Fatalf("query all: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0].Name != "Shanghai" || got[1].Name != "Hangzhou" {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

func TestCloseDBIsSafeWithoutInit(t *testing.T) {
	CloseDB()
}

func TestInitDBCreatesFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "created.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer CloseDB()

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}
