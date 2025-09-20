package handlers

import (
	"context"
	"denchokun-api/models"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type QueryRequest struct {
	Period     string                 `json:"period" binding:"required"`
	Query      string                 `json:"query" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
	Limit      int                    `json:"limit"`
}

type QueryResponse struct {
	Success bool                     `json:"success"`
	Columns []string                 `json:"columns,omitempty"`
	Rows    []map[string]interface{} `json:"rows,omitempty"`
	Count   int                      `json:"count,omitempty"`
	Error   string                   `json:"error,omitempty"`
	Message string                   `json:"message,omitempty"`
}

var (
	// 禁止するSQLキーワード（単語境界を考慮）
	forbiddenKeywords = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|TRUNCATE|EXEC|EXECUTE|GRANT|REVOKE|UNION|INTO|OUTFILE|DUMPFILE|LOAD_FILE|BENCHMARK|SLEEP|WAITFOR|PRAGMA|ATTACH|DETACH)\b`)
	
	// コメントパターン
	commentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`--.*$`),
		regexp.MustCompile(`/\*.*?\*/`),
		regexp.MustCompile(`#.*$`),
	}
)

// validateQuery SQLクエリの安全性を検証
func validateQuery(query string) error {
	// 空白を正規化
	query = strings.TrimSpace(query)
	
	// 空のクエリをチェック
	if query == "" {
		return fmt.Errorf("empty query")
	}
	
	// コメントを除去
	for _, pattern := range commentPatterns {
		query = pattern.ReplaceAllString(query, "")
	}
	
	// SELECT文で始まることを確認
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
		return fmt.Errorf("only SELECT statements are allowed")
	}
	
	// 禁止キーワードのチェック
	if forbiddenKeywords.MatchString(query) {
		return fmt.Errorf("forbidden SQL keyword detected")
	}
	
	// セミコロンで複数のステートメントを実行しようとしていないか確認
	if strings.Count(query, ";") > 0 {
		// 末尾のセミコロンは許可
		trimmed := strings.TrimRight(query, "; \t\n")
		if strings.Contains(trimmed, ";") {
			return fmt.Errorf("multiple statements are not allowed")
		}
	}
	
	// システムテーブルへのアクセスを制限
	lowerQuery := strings.ToLower(query)
	systemTables := []string{"sqlite_master", "sqlite_temp_master", "sqlite_sequence"}
	for _, table := range systemTables {
		if strings.Contains(lowerQuery, table) {
			return fmt.Errorf("access to system tables is not allowed")
		}
	}
	
	return nil
}

// ExecuteQuery SELECT専用のSQL実行API
func ExecuteQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}
	
	// クエリの検証
	if err := validateQuery(req.Query); err != nil {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Error:   "invalid_query",
			Message: err.Error(),
		})
		return
	}
	
	// デフォルトのリミット設定
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 100
	}
	
	// LIMITが指定されていない場合は自動的に追加
	upperQuery := strings.ToUpper(req.Query)
	if !strings.Contains(upperQuery, "LIMIT") {
		req.Query = fmt.Sprintf("%s LIMIT %d", req.Query, req.Limit)
	}
	
	// データベース接続を取得
	db, err := models.ConnectPeriodDB(req.Period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, QueryResponse{
			Success: false,
			Error:   "database_error",
			Message: fmt.Sprintf("Failed to connect to period %s: %v", req.Period, err),
		})
		return
	}
	
	// タイムアウト設定（30秒）
	ctx := c.Request.Context()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// クエリ実行
	rows, err := db.QueryContext(ctx, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, QueryResponse{
			Success: false,
			Error:   "query_execution_error",
			Message: err.Error(),
		})
		return
	}
	defer rows.Close()
	
	// カラム情報を取得
	columns, err := rows.Columns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, QueryResponse{
			Success: false,
			Error:   "column_error",
			Message: err.Error(),
		})
		return
	}
	
	// 結果を格納
	var results []map[string]interface{}
	
	// 各行を処理
	for rows.Next() {
		// カラム数に応じてスキャン用のスライスを作成
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		// 行をスキャン
		if err := rows.Scan(valuePtrs...); err != nil {
			c.JSON(http.StatusInternalServerError, QueryResponse{
				Success: false,
				Error:   "scan_error",
				Message: err.Error(),
			})
			return
		}
		
		// マップに変換
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			
			// NULL値の処理
			if val == nil {
				row[col] = nil
				continue
			}
			
			// バイト配列は文字列に変換
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		
		results = append(results, row)
	}
	
	// エラーチェック
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, QueryResponse{
			Success: false,
			Error:   "iteration_error",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, QueryResponse{
		Success: true,
		Columns: columns,
		Rows:    results,
		Count:   len(results),
	})
}