package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// QueryResult 查询结果
type QueryResult struct {
	Type       string                     `json:"type"`
	Data       []map[string]interface{}   `json:"data,omitempty"`
	ResultSets [][]map[string]interface{} `json:"result_sets,omitempty"` // 多结果集支持
	Count      int                        `json:"count,omitempty"`
	Offset     int                        `json:"offset,omitempty"`
	HasMore    bool                       `json:"has_more,omitempty"`
	NextOffset int                        `json:"next_offset,omitempty"`
	Truncated  bool                       `json:"truncated,omitempty"`
	Success    bool                       `json:"success,omitempty"`
	Message    string                     `json:"message,omitempty"`
}

// ExecResult 执行结果
type ExecResult struct {
	Type         string `json:"type"`
	Success      bool   `json:"success"`
	RowsAffected int64  `json:"rows_affected,omitempty"`
	LastInsertID int64  `json:"last_insert_id,omitempty"`
	Message      string `json:"message,omitempty"`
}

// Query 执行查询操作 (SELECT)
func Query(sql string, args ...interface{}) (*QueryResult, error) {
	return QueryContext(context.Background(), sql, 0, 0, args...)
}

// QueryContext 执行带上下文和行数限制的查询。
func QueryContext(ctx context.Context, sql string, offset, maxRows int, args ...interface{}) (*QueryResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	rawClient := GetRawDB()
	if rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := rawClient.QueryContext(ctx, sql, args...)
	if err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}
	defer rows.Close()

	results, hasMore, err := scanRows(rows, offset, maxRows)
	if err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	nextOffset := 0
	if hasMore {
		nextOffset = offset + len(results)
	}

	return &QueryResult{
		Type:       "select",
		Data:       results,
		Count:      len(results),
		Offset:     offset,
		HasMore:    hasMore,
		NextOffset: nextOffset,
		Truncated:  hasMore,
		Success:    true,
	}, nil
}

// Exec 执行操作 (INSERT/UPDATE/DELETE)
func Exec(sql string, args ...interface{}) (*ExecResult, error) {
	return ExecContext(context.Background(), sql, args...)
}

// ExecContext 执行带上下文的写操作。
func ExecContext(ctx context.Context, sql string, args ...interface{}) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	rawClient := GetRawDB()
	if rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	sqlTrimmed := strings.TrimSpace(strings.ToUpper(sql))
	if strings.HasPrefix(sqlTrimmed, "INSERT") {
		result, err := rawClient.ExecContext(ctx, sql, args...)
		if err != nil {
			return &ExecResult{
				Type:    "error",
				Success: false,
				Message: err.Error(),
			}, nil
		}

		lastID, err := result.LastInsertId()
		if err != nil {
			return &ExecResult{
				Type:    "error",
				Success: false,
				Message: fmt.Sprintf("failed to get last insert ID: %v", err),
			}, nil
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return &ExecResult{
				Type:    "error",
				Success: false,
				Message: fmt.Sprintf("failed to get rows affected: %v", err),
			}, nil
		}

		return &ExecResult{
			Type:         "insert",
			Success:      true,
			RowsAffected: rowsAffected,
			LastInsertID: lastID,
		}, nil
	}

	result, err := rawClient.ExecContext(ctx, sql, args...)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("failed to get rows affected: %v", err),
		}, nil
	}

	return &ExecResult{
		Type:         "modification",
		Success:      true,
		RowsAffected: rowsAffected,
	}, nil
}

// ExecWithLastID 执行INSERT操作并返回最后插入的ID
func ExecWithLastID(sql string, args ...interface{}) (*ExecResult, error) {
	return ExecWithLastIDContext(context.Background(), sql, args...)
}

// ExecWithLastIDContext 执行带上下文的INSERT并返回最后插入ID。
func ExecWithLastIDContext(ctx context.Context, sql string, args ...interface{}) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	rawClient := GetRawDB()
	if rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	result, err := rawClient.ExecContext(ctx, sql, args...)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("failed to get last insert ID: %v", err),
		}, nil
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("failed to get rows affected: %v", err),
		}, nil
	}

	return &ExecResult{
		Type:         "insert",
		Success:      true,
		LastInsertID: lastID,
		RowsAffected: rowsAffected,
	}, nil
}

// CallProcedure 调用存储过程，支持动态数量的结果集
func CallProcedure(procName string, args ...interface{}) (*QueryResult, error) {
	return CallProcedureContext(context.Background(), procName, 0, args...)
}

// CallProcedureContext 调用带上下文和行数限制的存储过程。
func CallProcedureContext(ctx context.Context, procName string, maxRows int, args ...interface{}) (*QueryResult, error) {
	rawClient := GetRawDB()
	if !IsConnected() || rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// 使用通用的多结果集处理方式
	return callProcedureGeneric(ctx, procName, maxRows, args...)
}

// callProcedureGeneric 通用的存储过程调用，支持任意数量的结果集
func callProcedureGeneric(ctx context.Context, procName string, maxRows int, args ...interface{}) (*QueryResult, error) {
	rawClient := GetRawDB()
	if rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// 构建CALL语句
	placeholders := make([]string, len(args))
	for i := range args {
		placeholders[i] = "?"
	}
	sql := fmt.Sprintf("CALL %s(%s)", procName, strings.Join(placeholders, ","))

	// 使用原始的database/sql来处理多个结果集，绕过zmysql的限制
	rows, err := rawClient.QueryContext(ctx, sql, args...)
	if err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}
	defer rows.Close()

	var allResultSets [][]map[string]interface{}
	anyTruncated := false

	// 处理多个结果集
	for {
		// 获取当前结果集的列信息
		currentResultSet, truncated, err := scanRows(rows, 0, maxRows)
		if err != nil {
			// 如果无法读取列信息，且已经有结果集，则视为结束。
			if len(allResultSets) > 0 && strings.Contains(err.Error(), "failed to get columns") {
				break
			}
			return &QueryResult{
				Type:    "error",
				Success: false,
				Message: err.Error(),
			}, nil
		}
		if truncated {
			anyTruncated = true
		}

		// 将当前结果集添加到所有结果集中
		allResultSets = append(allResultSets, currentResultSet)

		// 检查是否还有更多结果集
		if !rows.NextResultSet() {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("row iteration error: %v", err),
		}, nil
	}

	// 计算总记录数
	totalRecords := 0
	for _, resultSet := range allResultSets {
		totalRecords += len(resultSet)
	}

	// 根据结果集数量决定返回格式
	if len(allResultSets) == 1 {
		// 单结果集：使用data字段，确保ResultSets为nil
		return &QueryResult{
			Type:       "procedure",
			Data:       allResultSets[0],
			ResultSets: nil, // 显式设置为nil
			Count:      1,
			Truncated:  anyTruncated,
			Success:    true,
			Message:    fmt.Sprintf("Successfully executed procedure with 1 result set. Total records: %d", totalRecords),
		}, nil
	} else {
		// 多结果集：使用result_sets字段，确保Data为nil
		return &QueryResult{
			Type:       "procedure",
			Data:       nil, // 显式设置为nil
			ResultSets: allResultSets,
			Count:      len(allResultSets),
			Truncated:  anyTruncated,
			Success:    true,
			Message:    fmt.Sprintf("Successfully executed procedure with %d result sets. Total records: %d", len(allResultSets), totalRecords),
		}, nil
	}
}

// CreateProcedure 创建存储过程
func CreateProcedure(procedureSQL string) (*ExecResult, error) {
	return CreateProcedureContext(context.Background(), procedureSQL)
}

// CreateProcedureContext 创建带上下文的存储过程。
func CreateProcedureContext(ctx context.Context, procedureSQL string) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	rawClient := GetRawDB()
	if rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	_, err := rawClient.ExecContext(ctx, procedureSQL)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ExecResult{
		Type:    "create_procedure",
		Success: true,
		Message: "Procedure created successfully",
	}, nil
}

// DropProcedure 删除存储过程
func DropProcedure(procName string) (*ExecResult, error) {
	return DropProcedureContext(context.Background(), procName)
}

// DropProcedureContext 删除带上下文的存储过程。
func DropProcedureContext(ctx context.Context, procName string) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	rawClient := GetRawDB()
	if rawClient == nil {
		return nil, fmt.Errorf("database not connected")
	}

	sql := fmt.Sprintf("DROP PROCEDURE IF EXISTS `%s`", procName)
	_, err := rawClient.ExecContext(ctx, sql)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ExecResult{
		Type:    "drop_procedure",
		Success: true,
		Message: fmt.Sprintf("Procedure %s dropped successfully", procName),
	}, nil
}

// ShowProcedures 显示存储过程列表
func ShowProcedures(databaseName string) (*QueryResult, error) {
	return ShowProceduresContext(context.Background(), databaseName, 0, 0)
}

// ShowProceduresContext 显示存储过程列表，支持上下文和行数限制。
func ShowProceduresContext(ctx context.Context, databaseName string, offset, maxRows int) (*QueryResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	sql := "SELECT ROUTINE_NAME as name, ROUTINE_TYPE as type, CREATED as created, LAST_ALTERED as last_altered FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_SCHEMA = ?"
	return QueryContext(ctx, sql, offset, maxRows, databaseName)
}

// CreateTable 创建表。
func CreateTable(sql string) (*ExecResult, error) {
	return Exec(sql)
}

// AlterTable 修改表结构。
func AlterTable(sql string) (*ExecResult, error) {
	return Exec(sql)
}

// DropTable 删除表。
func DropTable(tableName string) (*ExecResult, error) {
	return Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
}

// ShowTables 列出当前数据库中的表。
func ShowTables() (*QueryResult, error) {
	return Query("SHOW TABLES")
}

// DescribeTable 返回表结构。
func DescribeTable(tableName string) (*QueryResult, error) {
	return Query(fmt.Sprintf("DESCRIBE `%s`", tableName))
}

// ShowCreateTable 返回建表语句。
func ShowCreateTable(tableName string) (*QueryResult, error) {
	return Query(fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName))
}

func scanRows(rows *sql.Rows, offset, maxRows int) ([]map[string]interface{}, bool, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	hasMore := false
	skipped := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, false, fmt.Errorf("failed to scan row: %w", err)
		}

		if skipped < offset {
			skipped++
			continue
		}

		if maxRows > 0 && len(results) >= maxRows {
			hasMore = true
			continue
		}

		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("row iteration error: %w", err)
	}

	return results, hasMore, nil
}
