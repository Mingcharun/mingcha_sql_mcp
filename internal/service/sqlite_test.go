package service

import "testing"

func TestIsSQLiteExecStatement(t *testing.T) {
	tests := []struct {
		sql      string
		expected bool
	}{
		{sql: "SELECT * FROM users", expected: false},
		{sql: " INSERT INTO users(name) VALUES (?)", expected: true},
		{sql: "update users set name = 'a'", expected: true},
		{sql: "CREATE TABLE demo(id integer)", expected: true},
		{sql: "pragma table_info(users)", expected: false},
		{sql: "VACUUM", expected: true},
	}

	for _, tt := range tests {
		got := isSQLiteExecStatement(tt.sql)
		if got != tt.expected {
			t.Fatalf("sql %q expected %v, got %v", tt.sql, tt.expected, got)
		}
	}
}
