package projectconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	mysqldb "github.com/Mingcharun/database-mcp/internal/database/mysql"
	postgresdb "github.com/Mingcharun/database-mcp/internal/database/postgres"
	redisdb "github.com/Mingcharun/database-mcp/internal/database/redis"
	"gopkg.in/yaml.v3"
)

const maxConfigFileSize = 1024 * 1024

type Source struct {
	FilePath string `json:"file_path"`
	Format   string `json:"format"`
}

type MySQLMatch struct {
	Config mysqldb.ConnectionConfig `json:"config"`
	Source Source                   `json:"source"`
}

type PostgresMatch struct {
	Config postgresdb.Config `json:"config"`
	Source Source            `json:"source"`
}

type RedisMatch struct {
	Config redisdb.Config `json:"config"`
	Source Source         `json:"source"`
}

type SQLiteMatch struct {
	DBPath string `json:"db_path"`
	Source Source `json:"source"`
}

type DetectionResult struct {
	ProjectPath   string         `json:"project_path"`
	SearchedFiles []string       `json:"searched_files"`
	MySQL         *MySQLMatch    `json:"mysql,omitempty"`
	Postgres      *PostgresMatch `json:"postgres,omitempty"`
	Redis         *RedisMatch    `json:"redis,omitempty"`
	SQLite        *SQLiteMatch   `json:"sqlite,omitempty"`
}

type parsedFile struct {
	source Source
	flat   map[string]interface{}
}

func Detect(projectPath, configFile string) (*DetectionResult, error) {
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}

	files, err := discoverConfigFiles(absProjectPath, configFile)
	if err != nil {
		return nil, err
	}

	envValues, err := collectEnvValues(files)
	if err != nil {
		return nil, err
	}

	parsedFiles := make([]parsedFile, 0, len(files))
	for _, filePath := range files {
		parsed, err := parseConfigFile(filePath, envValues)
		if err != nil {
			continue
		}
		parsedFiles = append(parsedFiles, parsed)
	}

	result := &DetectionResult{
		ProjectPath:   absProjectPath,
		SearchedFiles: files,
	}

	for _, file := range parsedFiles {
		if result.MySQL == nil {
			if match, ok := detectMySQL(file); ok {
				result.MySQL = match
			}
		}
		if result.Postgres == nil {
			if match, ok := detectPostgres(file); ok {
				result.Postgres = match
			}
		}
		if result.Redis == nil {
			if match, ok := detectRedis(file); ok {
				result.Redis = match
			}
		}
		if result.SQLite == nil {
			if match, ok := detectSQLite(file); ok {
				result.SQLite = match
			}
		}
	}

	return result, nil
}

func ResolveMySQL(projectPath, configFile string) (*MySQLMatch, error) {
	result, err := Detect(projectPath, configFile)
	if err != nil {
		return nil, err
	}
	if result.MySQL == nil {
		return nil, fmt.Errorf("no MySQL config detected in %s", strings.Join(result.SearchedFiles, ", "))
	}
	return result.MySQL, nil
}

func ResolvePostgres(projectPath, configFile string) (*PostgresMatch, error) {
	result, err := Detect(projectPath, configFile)
	if err != nil {
		return nil, err
	}
	if result.Postgres == nil {
		return nil, fmt.Errorf("no PostgreSQL config detected in %s", strings.Join(result.SearchedFiles, ", "))
	}
	return result.Postgres, nil
}

func ResolveRedis(projectPath, configFile string) (*RedisMatch, error) {
	result, err := Detect(projectPath, configFile)
	if err != nil {
		return nil, err
	}
	if result.Redis == nil {
		return nil, fmt.Errorf("no Redis config detected in %s", strings.Join(result.SearchedFiles, ", "))
	}
	return result.Redis, nil
}

func ResolveSQLite(projectPath, configFile string) (*SQLiteMatch, error) {
	result, err := Detect(projectPath, configFile)
	if err != nil {
		return nil, err
	}
	if result.SQLite == nil {
		return nil, fmt.Errorf("no SQLite config detected in %s", strings.Join(result.SearchedFiles, ", "))
	}
	return result.SQLite, nil
}

func discoverConfigFiles(projectPath, configFile string) ([]string, error) {
	if configFile != "" {
		filePath := configFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(projectPath, configFile)
		}
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("stat config file: %w", err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("config_file must be a file, got directory: %s", filePath)
		}
		return []string{filePath}, nil
	}

	var files []string
	err := filepath.WalkDir(projectPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != projectPath {
				rel, relErr := filepath.Rel(projectPath, path)
				if relErr == nil && pathDepth(rel) > 2 {
					return filepath.SkipDir
				}
			}
			name := d.Name()
			switch name {
			case ".git", "node_modules", "vendor", "dist", "build", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		if !looksLikeConfigFile(path) {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil || info.Size() > maxConfigFileSize {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		leftPriority := configPriority(files[i])
		rightPriority := configPriority(files[j])
		if leftPriority == rightPriority {
			return files[i] < files[j]
		}
		return leftPriority < rightPriority
	})

	if len(files) == 0 {
		return nil, fmt.Errorf("no supported config files found under %s", projectPath)
	}
	return files, nil
}

func collectEnvValues(files []string) (map[string]string, error) {
	values := make(map[string]string)
	for _, filePath := range files {
		if detectFormat(filePath) != "env" {
			continue
		}
		parsed, err := parseEnvFile(filePath, values)
		if err != nil {
			return nil, err
		}
		for key, value := range parsed {
			values[key] = value
		}
	}
	return values, nil
}

func parseConfigFile(filePath string, envValues map[string]string) (parsedFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return parsedFile{}, err
	}

	format := detectFormat(filePath)
	switch format {
	case "env":
		envMap, err := parseEnvFile(filePath, envValues)
		if err != nil {
			return parsedFile{}, err
		}
		flat := make(map[string]interface{}, len(envMap))
		for key, value := range envMap {
			flat[strings.ToLower(key)] = value
		}
		return parsedFile{source: Source{FilePath: filePath, Format: format}, flat: flat}, nil
	case "json":
		var raw interface{}
		if err := json.Unmarshal(content, &raw); err != nil {
			return parsedFile{}, err
		}
		return parsedFile{source: Source{FilePath: filePath, Format: format}, flat: flattenValue(expandValue(normalizeValue(raw), envValues))}, nil
	case "yaml":
		var raw interface{}
		if err := yaml.Unmarshal(content, &raw); err != nil {
			return parsedFile{}, err
		}
		return parsedFile{source: Source{FilePath: filePath, Format: format}, flat: flattenValue(expandValue(normalizeValue(raw), envValues))}, nil
	case "toml":
		raw := make(map[string]interface{})
		if err := toml.Unmarshal(content, &raw); err != nil {
			return parsedFile{}, err
		}
		return parsedFile{source: Source{FilePath: filePath, Format: format}, flat: flattenValue(expandValue(normalizeValue(raw), envValues))}, nil
	case "properties":
		raw, err := parseProperties(content, envValues)
		if err != nil {
			return parsedFile{}, err
		}
		flat := make(map[string]interface{}, len(raw))
		for key, value := range raw {
			flat[strings.ToLower(key)] = value
		}
		return parsedFile{source: Source{FilePath: filePath, Format: format}, flat: flat}, nil
	default:
		return parsedFile{}, fmt.Errorf("unsupported config format for %s", filePath)
	}
}

func detectMySQL(file parsedFile) (*MySQLMatch, bool) {
	if cfg, ok := mysqlFromURL(file); ok {
		return &MySQLMatch{Config: cfg, Source: file.source}, true
	}

	host := lookupString(file.flat,
		"mysql.host", "mysql_host",
		"database.mysql.host", "databases.mysql.host",
		"db.host", "db_host",
		"spring.datasource.host",
	)
	if host == "" {
		return nil, false
	}

	cfg := mysqldb.ConnectionConfig{
		Username: lookupString(file.flat,
			"mysql.username", "mysql.user", "mysql_username", "mysql_user",
			"database.mysql.username", "database.mysql.user",
			"db.username", "db.user", "db_username", "db_user",
			"spring.datasource.username",
		),
		Password: lookupString(file.flat,
			"mysql.password", "mysql_password",
			"database.mysql.password",
			"db.password", "db_password",
			"spring.datasource.password",
		),
		Addr: net.JoinHostPort(host, lookupPortString(file.flat, 3306,
			"mysql.port", "mysql_port",
			"database.mysql.port",
			"db.port", "db_port",
			"spring.datasource.port",
		)),
		DatabaseName: lookupString(file.flat,
			"mysql.database", "mysql.name", "mysql.dbname", "mysql.database_name",
			"mysql_database", "mysql_dbname",
			"database.mysql.database", "database.mysql.name", "database.mysql.dbname",
			"db.database", "db.name", "db.dbname", "db.database_name",
			"db_database", "db_name",
		),
	}

	if cfg.Username == "" || cfg.DatabaseName == "" {
		return nil, false
	}
	return &MySQLMatch{Config: cfg, Source: file.source}, true
}

func detectPostgres(file parsedFile) (*PostgresMatch, bool) {
	if cfg, ok := postgresFromURL(file); ok {
		return &PostgresMatch{Config: cfg, Source: file.source}, true
	}

	host := lookupString(file.flat,
		"postgres.host", "postgres_host",
		"pgsql.host", "pgsql_host",
		"database.postgres.host", "database.pgsql.host",
		"db.host", "db_host",
	)
	if host == "" {
		return nil, false
	}

	cfg := postgresdb.Config{
		Host: host,
		Port: lookupPortInt(file.flat, 5432,
			"postgres.port", "postgres_port",
			"pgsql.port", "pgsql_port",
			"database.postgres.port", "database.pgsql.port",
			"db.port", "db_port",
		),
		User: lookupString(file.flat,
			"postgres.username", "postgres.user", "postgres_username", "postgres_user",
			"pgsql.username", "pgsql.user", "pgsql_username", "pgsql_user",
			"database.postgres.username", "database.postgres.user",
			"db.username", "db.user", "db_username", "db_user",
			"spring.datasource.username",
		),
		Password: lookupString(file.flat,
			"postgres.password", "postgres_password",
			"pgsql.password", "pgsql_password",
			"database.postgres.password",
			"db.password", "db_password",
			"spring.datasource.password",
		),
		Database: lookupString(file.flat,
			"postgres.database", "postgres.dbname", "postgres_database",
			"pgsql.database", "pgsql.dbname", "pgsql_database",
			"database.postgres.database", "database.postgres.dbname",
			"db.database", "db.name", "db.dbname", "db_database", "db_name",
		),
		SSLMode: lookupString(file.flat,
			"postgres.sslmode", "postgres_sslmode",
			"pgsql.sslmode", "pgsql_sslmode",
			"database.postgres.sslmode",
			"db.sslmode", "db_sslmode",
		),
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.User == "" || cfg.Database == "" {
		return nil, false
	}
	return &PostgresMatch{Config: cfg, Source: file.source}, true
}

func detectRedis(file parsedFile) (*RedisMatch, bool) {
	if cfg, ok := redisFromURL(file); ok {
		return &RedisMatch{Config: cfg, Source: file.source}, true
	}

	host := lookupString(file.flat,
		"redis.host", "redis_host",
		"spring.data.redis.host",
		"cache.redis.host", "cache.redis_host",
	)
	if host == "" {
		return nil, false
	}

	cfg := redisdb.Config{
		Addr: net.JoinHostPort(host, lookupPortString(file.flat, 6379,
			"redis.port", "redis_port",
			"spring.data.redis.port",
			"cache.redis.port", "cache.redis_port",
		)),
		Password: lookupString(file.flat,
			"redis.password", "redis_password",
			"spring.data.redis.password",
			"cache.redis.password", "cache.redis_password",
		),
		DB: lookupPortInt(file.flat, 0,
			"redis.database", "redis.db", "redis_database", "redis_db",
			"spring.data.redis.database",
			"cache.redis.database", "cache.redis.db",
		),
	}
	return &RedisMatch{Config: cfg, Source: file.source}, true
}

func detectSQLite(file parsedFile) (*SQLiteMatch, bool) {
	for _, key := range []string{
		"sqlite.path", "sqlite.file", "sqlite.db", "sqlite_path", "sqlite_file", "sqlite_db",
		"database.sqlite.path", "database.sqlite.file",
		"db.sqlite.path", "db.sqlite.file",
		"database.path", "database.file",
	} {
		if rawPath := lookupString(file.flat, key); rawPath != "" {
			return &SQLiteMatch{
				DBPath: resolveRelativeFilePath(file.source.FilePath, rawPath),
				Source: file.source,
			}, true
		}
	}

	for _, key := range []string{"sqlite.url", "sqlite_url", "database.url", "database_url", "datasource.url", "spring.datasource.url"} {
		rawValue := lookupString(file.flat, key)
		if rawValue == "" {
			continue
		}
		if parsed, ok := parseDatabaseURL(rawValue); ok && parsed.scheme == "sqlite" && parsed.path != "" {
			return &SQLiteMatch{
				DBPath: resolveRelativeFilePath(file.source.FilePath, parsed.path),
				Source: file.source,
			}, true
		}
	}

	return nil, false
}

func mysqlFromURL(file parsedFile) (mysqldb.ConnectionConfig, bool) {
	for _, key := range []string{
		"mysql.url", "mysql.dsn", "mysql_url", "mysql_dsn",
		"database.mysql.url", "database.mysql.dsn",
		"db.url", "db.dsn", "db_url", "db_dsn",
		"database.url", "database.dsn", "database_url", "database_dsn",
		"datasource.url", "datasource.dsn",
		"spring.datasource.url",
		"mysql.datasource.url",
	} {
		rawValue := lookupString(file.flat, key)
		if rawValue == "" {
			continue
		}
		parsed, ok := parseDatabaseURL(rawValue)
		if !ok || parsed.scheme != "mysql" {
			continue
		}
		cfg := mysqldb.ConnectionConfig{
			Username:     coalesce(parsed.user, lookupString(file.flat, "spring.datasource.username", "mysql.username", "db.username")),
			Password:     coalesce(parsed.password, lookupString(file.flat, "spring.datasource.password", "mysql.password", "db.password")),
			Addr:         net.JoinHostPort(parsed.host, coalesce(parsed.port, "3306")),
			DatabaseName: strings.TrimPrefix(parsed.database, "/"),
		}
		if cfg.Username == "" || cfg.DatabaseName == "" || parsed.host == "" {
			continue
		}
		return cfg, true
	}
	return mysqldb.ConnectionConfig{}, false
}

func postgresFromURL(file parsedFile) (postgresdb.Config, bool) {
	for _, key := range []string{
		"postgres.url", "postgres_url",
		"pgsql.url", "pgsql_url",
		"database.postgres.url", "database.pgsql.url",
		"db.url", "db_url",
		"database.url", "database_url",
		"datasource.url",
		"spring.datasource.url",
	} {
		rawValue := lookupString(file.flat, key)
		if rawValue == "" {
			continue
		}
		parsed, ok := parseDatabaseURL(rawValue)
		if !ok || (parsed.scheme != "postgres" && parsed.scheme != "postgresql") {
			continue
		}
		cfg := postgresdb.Config{
			Host:     parsed.host,
			Port:     mustInt(coalesce(parsed.port, "5432"), 5432),
			User:     coalesce(parsed.user, lookupString(file.flat, "spring.datasource.username", "postgres.username", "db.username")),
			Password: coalesce(parsed.password, lookupString(file.flat, "spring.datasource.password", "postgres.password", "db.password")),
			Database: strings.TrimPrefix(parsed.database, "/"),
			SSLMode:  coalesce(parsed.query.Get("sslmode"), lookupString(file.flat, "postgres.sslmode", "db.sslmode"), "disable"),
		}
		if cfg.Host == "" || cfg.User == "" || cfg.Database == "" {
			continue
		}
		return cfg, true
	}
	return postgresdb.Config{}, false
}

func redisFromURL(file parsedFile) (redisdb.Config, bool) {
	for _, key := range []string{
		"redis.url", "redis_url",
		"spring.data.redis.url",
		"cache.redis.url", "cache.redis_url",
	} {
		rawValue := lookupString(file.flat, key)
		if rawValue == "" {
			continue
		}
		parsed, ok := parseDatabaseURL(rawValue)
		if !ok || (parsed.scheme != "redis" && parsed.scheme != "rediss") {
			continue
		}
		cfg := redisdb.Config{
			Addr:     net.JoinHostPort(parsed.host, coalesce(parsed.port, "6379")),
			Password: parsed.password,
			DB:       mustInt(strings.TrimPrefix(parsed.database, "/"), 0),
		}
		if dbQuery := parsed.query.Get("db"); dbQuery != "" {
			cfg.DB = mustInt(dbQuery, cfg.DB)
		}
		if parsed.scheme == "rediss" {
			skipVerify := false
			cfg.SSLInsecureSkipVerify = &skipVerify
		}
		if parsed.host == "" {
			continue
		}
		return cfg, true
	}
	return redisdb.Config{}, false
}

type parsedURL struct {
	scheme   string
	host     string
	port     string
	user     string
	password string
	database string
	path     string
	query    url.Values
}

var mysqlDSNPattern = regexp.MustCompile(`^([^:]+):([^@]*)@tcp\(([^)]+)\)/([^?]+)`)

func parseDatabaseURL(rawValue string) (parsedURL, bool) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return parsedURL{}, false
	}

	if matches := mysqlDSNPattern.FindStringSubmatch(trimmed); len(matches) == 5 {
		host, port, err := net.SplitHostPort(matches[3])
		if err != nil {
			host = matches[3]
			port = "3306"
		}
		return parsedURL{
			scheme:   "mysql",
			host:     host,
			port:     port,
			user:     matches[1],
			password: matches[2],
			database: matches[4],
		}, true
	}

	if strings.HasPrefix(trimmed, "jdbc:") {
		trimmed = strings.TrimPrefix(trimmed, "jdbc:")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return parsedURL{}, false
	}
	if parsed.Scheme == "" {
		return parsedURL{}, false
	}

	password, _ := parsed.User.Password()
	return parsedURL{
		scheme:   strings.ToLower(parsed.Scheme),
		host:     parsed.Hostname(),
		port:     parsed.Port(),
		user:     parsed.User.Username(),
		password: password,
		database: strings.TrimPrefix(parsed.Path, "/"),
		path:     parsed.Path,
		query:    parsed.Query(),
	}, true
}

func parseEnvFile(filePath string, envValues map[string]string) (map[string]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		separator := strings.Index(line, "=")
		if separator < 0 {
			continue
		}
		key := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		value = trimWrappingQuotes(value)
		value = expandString(value, mergeEnvMaps(envValues, result))
		result[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func parseProperties(content []byte, envValues map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		separator := strings.IndexAny(line, "=:")
		if separator < 0 {
			continue
		}
		key := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		result[key] = expandString(trimWrappingQuotes(value), envValues)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func flattenValue(value interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	flattenInto("", value, result)
	return result
}

func flattenInto(prefix string, value interface{}, output map[string]interface{}) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			nextKey := strings.ToLower(key)
			if prefix != "" {
				nextKey = prefix + "." + nextKey
			}
			flattenInto(nextKey, child, output)
		}
	case []interface{}:
		for index, child := range typed {
			nextKey := fmt.Sprintf("%s.%d", prefix, index)
			flattenInto(nextKey, child, output)
		}
	default:
		if prefix != "" {
			output[strings.ToLower(prefix)] = typed
		}
	}
}

func normalizeValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			result[key] = normalizeValue(child)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			result[fmt.Sprint(key)] = normalizeValue(child)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(typed))
		for index, child := range typed {
			result[index] = normalizeValue(child)
		}
		return result
	default:
		return typed
	}
}

func expandValue(value interface{}, envValues map[string]string) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			result[key] = expandValue(child, envValues)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(typed))
		for index, child := range typed {
			result[index] = expandValue(child, envValues)
		}
		return result
	case string:
		return expandString(typed, envValues)
	default:
		return typed
	}
}

var placeholderPattern = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

func expandString(value string, envValues map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := placeholderPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		key := parts[1]
		if resolved, ok := envValues[key]; ok && resolved != "" {
			return resolved
		}
		if resolved := os.Getenv(key); resolved != "" {
			return resolved
		}
		if len(parts) > 2 && parts[2] != "" {
			return parts[2]
		}
		return ""
	})
}

func mergeEnvMaps(base map[string]string, overlay map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func lookupString(flat map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := flat[strings.ToLower(key)]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case int:
			return strconv.Itoa(typed)
		case int64:
			return strconv.FormatInt(typed, 10)
		case float64:
			return strconv.Itoa(int(typed))
		case bool:
			return strconv.FormatBool(typed)
		}
	}
	return ""
}

func lookupPortString(flat map[string]interface{}, defaultPort int, keys ...string) string {
	return strconv.Itoa(lookupPortInt(flat, defaultPort, keys...))
}

func lookupPortInt(flat map[string]interface{}, defaultPort int, keys ...string) int {
	for _, key := range keys {
		value, ok := flat[strings.ToLower(key)]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		case float64:
			return int(typed)
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
				return parsed
			}
		}
	}
	return defaultPort
}

func mustInt(value string, defaultValue int) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return parsed
	}
	return defaultValue
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func trimWrappingQuotes(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func resolveRelativeFilePath(configFilePath, dbPath string) string {
	if dbPath == "" {
		return dbPath
	}
	if filepath.IsAbs(dbPath) {
		return filepath.Clean(dbPath)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(configFilePath), dbPath))
}

func looksLikeConfigFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))

	if strings.HasPrefix(base, ".env") {
		return true
	}
	switch ext {
	case ".json", ".yaml", ".yml", ".toml", ".properties":
		return true
	}
	switch base {
	case "application.properties", "application.yml", "application.yaml", "config.yml", "config.yaml", "config.json", "config.toml", "settings.json":
		return true
	}
	return false
}

func detectFormat(path string) string {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	if strings.HasPrefix(base, ".env") {
		return "env"
	}
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".properties":
		return "properties"
	default:
		return ""
	}
}

func configPriority(path string) int {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case base == ".env.local":
		return 0
	case base == ".env":
		return 1
	case strings.HasPrefix(base, ".env"):
		return 2
	case base == "application.yml" || base == "application.yaml":
		return 3
	case base == "application.properties":
		return 4
	case base == "config.yml" || base == "config.yaml":
		return 5
	case base == "config.json":
		return 6
	case base == "config.toml":
		return 7
	case strings.HasSuffix(base, ".properties"):
		return 8
	default:
		return 9
	}
}

func pathDepth(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}
