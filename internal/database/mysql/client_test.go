package mysql

import (
	"os"
	"strings"
	"testing"
	"time"
)

func testMySQLConfig(t *testing.T) ConnectionConfig {
	t.Helper()

	config := ConnectionConfig{
		Username:        os.Getenv("MYSQL_TEST_USER"),
		Password:        os.Getenv("MYSQL_TEST_PASSWORD"),
		Addr:            os.Getenv("MYSQL_TEST_ADDR"),
		DatabaseName:    os.Getenv("MYSQL_TEST_DATABASE"),
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	}

	if config.Username == "" || config.Addr == "" || config.DatabaseName == "" {
		t.Skip("skipping MySQL integration tests; set MYSQL_TEST_USER, MYSQL_TEST_PASSWORD, MYSQL_TEST_ADDR, MYSQL_TEST_DATABASE")
	}

	return config
}

func withTestDB(t *testing.T, fn func(t *testing.T)) {
	t.Helper()

	if err := InitDB(testMySQLConfig(t)); err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer CloseDB()

	fn(t)
}

func TestMySQLExecQueryAndMetadata(t *testing.T) {
	withTestDB(t, func(t *testing.T) {
		t.Helper()

		if _, err := DropTable("test_users"); err != nil {
			t.Fatalf("drop test table: %v", err)
		}

		createSQL := `
			CREATE TABLE test_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE,
				age INT DEFAULT 0
			)
		`
		if _, err := CreateTable(createSQL); err != nil {
			t.Fatalf("create table: %v", err)
		}
		defer DropTable("test_users")

		insertResult, err := ExecWithLastID(
			"INSERT INTO test_users (name, email, age) VALUES (?, ?, ?)",
			"张三",
			"zhangsan@example.com",
			25,
		)
		if err != nil {
			t.Fatalf("insert row: %v", err)
		}
		if !insertResult.Success || insertResult.LastInsertID <= 0 {
			t.Fatalf("unexpected insert result: %+v", insertResult)
		}

		queryResult, err := Query("SELECT id, name, email, age FROM test_users WHERE id = ?", insertResult.LastInsertID)
		if err != nil {
			t.Fatalf("query row: %v", err)
		}
		if !queryResult.Success || queryResult.Count != 1 {
			t.Fatalf("unexpected query result: %+v", queryResult)
		}
		if got := queryResult.Data[0]["name"]; got != "张三" {
			t.Fatalf("unexpected name: %v", got)
		}

		updateResult, err := Exec("UPDATE test_users SET age = ? WHERE id = ?", 26, insertResult.LastInsertID)
		if err != nil {
			t.Fatalf("update row: %v", err)
		}
		if !updateResult.Success || updateResult.RowsAffected != 1 {
			t.Fatalf("unexpected update result: %+v", updateResult)
		}

		tables, err := ShowTables()
		if err != nil {
			t.Fatalf("show tables: %v", err)
		}
		if !tables.Success {
			t.Fatalf("unexpected show tables result: %+v", tables)
		}

		describeResult, err := DescribeTable("test_users")
		if err != nil {
			t.Fatalf("describe table: %v", err)
		}
		if !describeResult.Success || describeResult.Count == 0 {
			t.Fatalf("unexpected describe result: %+v", describeResult)
		}

		showCreateResult, err := ShowCreateTable("test_users")
		if err != nil {
			t.Fatalf("show create table: %v", err)
		}
		if !showCreateResult.Success || showCreateResult.Count != 1 {
			t.Fatalf("unexpected show create result: %+v", showCreateResult)
		}
	})
}

func TestMySQLProcedureMultipleResultSets(t *testing.T) {
	withTestDB(t, func(t *testing.T) {
		t.Helper()

		_, _ = DropProcedure("GetMultipleData")
		_, _ = DropTable("test_users")
		_, _ = DropTable("test_orders")

		if _, err := CreateTable(`
			CREATE TABLE test_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(100) NOT NULL
			)
		`); err != nil {
			t.Fatalf("create users table: %v", err)
		}
		defer DropTable("test_users")

		if _, err := CreateTable(`
			CREATE TABLE test_orders (
				id INT AUTO_INCREMENT PRIMARY KEY,
				user_id INT NOT NULL,
				product VARCHAR(100) NOT NULL
			)
		`); err != nil {
			t.Fatalf("create orders table: %v", err)
		}
		defer DropTable("test_orders")

		if _, err := ExecWithLastID("INSERT INTO test_users (name) VALUES (?)", "张三"); err != nil {
			t.Fatalf("insert user: %v", err)
		}
		if _, err := ExecWithLastID("INSERT INTO test_orders (user_id, product) VALUES (?, ?)", 1, "商品A"); err != nil {
			t.Fatalf("insert order: %v", err)
		}

		createProcSQL := `
			CREATE PROCEDURE GetMultipleData()
			BEGIN
				SELECT * FROM test_users;
				SELECT * FROM test_orders;
			END
		`
		if _, err := CreateProcedure(createProcSQL); err != nil {
			t.Fatalf("create procedure: %v", err)
		}
		defer DropProcedure("GetMultipleData")

		result, err := CallProcedure("GetMultipleData")
		if err != nil {
			t.Fatalf("call procedure: %v", err)
		}
		if !result.Success {
			t.Fatalf("procedure should succeed: %+v", result)
		}
		if len(result.ResultSets) != 2 {
			t.Fatalf("expected 2 result sets, got %d", len(result.ResultSets))
		}
	})
}

func TestShowCreateTableIncludesDefinition(t *testing.T) {
	withTestDB(t, func(t *testing.T) {
		t.Helper()

		_, _ = DropTable("test_users")
		if _, err := CreateTable(`
			CREATE TABLE test_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(100) NOT NULL
			)
		`); err != nil {
			t.Fatalf("create table: %v", err)
		}
		defer DropTable("test_users")

		result, err := ShowCreateTable("test_users")
		if err != nil {
			t.Fatalf("show create table: %v", err)
		}
		if result.Count != 1 {
			t.Fatalf("expected 1 row, got %+v", result)
		}

		foundDefinition := false
		for _, value := range result.Data[0] {
			if strings.Contains(strings.ToUpper(value.(string)), "CREATE TABLE") {
				foundDefinition = true
				break
			}
		}
		if !foundDefinition {
			t.Fatalf("expected CREATE TABLE definition in result: %+v", result.Data[0])
		}
	})
}
