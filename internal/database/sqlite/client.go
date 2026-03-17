package sqlite

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var (
	db *sqlx.DB
	mu sync.RWMutex
)

func DB() *sqlx.DB {
	mu.RLock()
	defer mu.RUnlock()
	return db
}
func InitDB(filePath string) error {
	CloseDB()

	var err error
	dsn := filePath
	client, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		log.Println("Error opening database:", err)
		return err
	}

	// SQLite 更适合保守的连接池，避免锁竞争。
	client.SetConnMaxLifetime(4 * time.Hour)
	client.SetMaxOpenConns(1)
	client.SetMaxIdleConns(1)
	err = client.Ping()
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("数据库连接失败ping: %w", err)
	}
	//_, err = db.Exec("PRAGMA journal_mode=WAL;")
	//if err != nil {
	//	log.Fatal(err)
	//}

	mu.Lock()
	db = client
	mu.Unlock()
	return nil
}
func CloseDB() {
	mu.Lock()
	currentDB := db
	db = nil
	mu.Unlock()

	if currentDB != nil {
		_ = currentDB.Close()
	}
}

type Scanner interface {
	Scan(dest ...interface{}) error
}

func QueryAll[T any](query string, destSlice *[]T, scanFunc func(row Scanner) T) error {
	client := DB()
	if client == nil {
		return fmt.Errorf("database not initialized")
	}

	rows, err := client.Query(query)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()
	//columns, err := rows.Columns()
	//if err != nil {
	//	log.Println(err, "Columns Error")
	//	return err
	//}
	var result []T
	for rows.Next() {
		tValue := scanFunc(rows)
		result = append(result, tValue)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over rows: %v", err)
	}

	*destSlice = result

	return nil
}
