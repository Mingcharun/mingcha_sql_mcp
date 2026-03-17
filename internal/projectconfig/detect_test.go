package projectconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFromEnvFile(t *testing.T) {
	projectDir := t.TempDir()
	envFile := filepath.Join(projectDir, ".env")
	if err := os.WriteFile(envFile, []byte(`
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3307
MYSQL_DATABASE=app_db
MYSQL_USERNAME=app_user
MYSQL_PASSWORD=secret
REDIS_HOST=127.0.0.1
REDIS_PORT=6380
REDIS_PASSWORD=redis-secret
REDIS_DB=2
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	result, err := Detect(projectDir, "")
	if err != nil {
		t.Fatalf("detect configs: %v", err)
	}

	if result.MySQL == nil {
		t.Fatal("expected MySQL config to be detected")
	}
	if result.MySQL.Config.Addr != "127.0.0.1:3307" {
		t.Fatalf("unexpected MySQL addr: %s", result.MySQL.Config.Addr)
	}
	if result.MySQL.Config.DatabaseName != "app_db" {
		t.Fatalf("unexpected MySQL database: %s", result.MySQL.Config.DatabaseName)
	}

	if result.Redis == nil {
		t.Fatal("expected Redis config to be detected")
	}
	if result.Redis.Config.Addr != "127.0.0.1:6380" {
		t.Fatalf("unexpected Redis addr: %s", result.Redis.Config.Addr)
	}
	if result.Redis.Config.DB != 2 {
		t.Fatalf("unexpected Redis DB: %d", result.Redis.Config.DB)
	}
}

func TestDetectSpringYAMLWithEnvPlaceholders(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte(`
DB_HOST=db.internal
DB_PORT=5433
DB_NAME=project_db
DB_USER=project_user
DB_PASSWORD=project_password
REDIS_HOST=cache.internal
REDIS_PORT=6381
`), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "application.yml"), []byte(`
spring:
  datasource:
    url: jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}
    username: ${DB_USER}
    password: ${DB_PASSWORD}
  data:
    redis:
      host: ${REDIS_HOST}
      port: ${REDIS_PORT}
`), 0o644); err != nil {
		t.Fatalf("write yaml file: %v", err)
	}

	result, err := Detect(projectDir, "")
	if err != nil {
		t.Fatalf("detect configs: %v", err)
	}

	if result.Postgres == nil {
		t.Fatal("expected PostgreSQL config to be detected")
	}
	if result.Postgres.Config.Host != "db.internal" || result.Postgres.Config.Port != 5433 {
		t.Fatalf("unexpected PostgreSQL host config: %+v", result.Postgres.Config)
	}
	if result.Postgres.Config.Database != "project_db" {
		t.Fatalf("unexpected PostgreSQL database: %s", result.Postgres.Config.Database)
	}

	if result.Redis == nil {
		t.Fatal("expected Redis config to be detected from YAML")
	}
	if result.Redis.Config.Addr != "cache.internal:6381" {
		t.Fatalf("unexpected Redis addr: %s", result.Redis.Config.Addr)
	}
}

func TestDetectSQLiteFromJSONConfig(t *testing.T) {
	projectDir := t.TempDir()
	configDir := filepath.Join(projectDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{
  "sqlite": {
    "path": "../data/app.sqlite"
  }
}`), 0o644); err != nil {
		t.Fatalf("write json file: %v", err)
	}

	result, err := Detect(projectDir, "")
	if err != nil {
		t.Fatalf("detect configs: %v", err)
	}

	if result.SQLite == nil {
		t.Fatal("expected SQLite config to be detected")
	}
	expectedPath := filepath.Clean(filepath.Join(configDir, "../data/app.sqlite"))
	if result.SQLite.DBPath != expectedPath {
		t.Fatalf("unexpected SQLite path: got %s want %s", result.SQLite.DBPath, expectedPath)
	}
}

func TestResolveMySQLFromSpecificConfigFile(t *testing.T) {
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, "custom.toml")
	if err := os.WriteFile(configPath, []byte(`
[mysql]
host = "localhost"
port = 3308
username = "toml_user"
password = "toml_secret"
database = "toml_db"
`), 0o644); err != nil {
		t.Fatalf("write toml file: %v", err)
	}

	match, err := ResolveMySQL(projectDir, "custom.toml")
	if err != nil {
		t.Fatalf("resolve MySQL: %v", err)
	}
	if match.Config.Addr != "localhost:3308" {
		t.Fatalf("unexpected addr: %s", match.Config.Addr)
	}
	if match.Config.Username != "toml_user" {
		t.Fatalf("unexpected username: %s", match.Config.Username)
	}
}
