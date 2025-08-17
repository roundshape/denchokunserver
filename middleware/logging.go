package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// カスタムResponseWriter - レスポンスボディをキャプチャするため
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggingMiddleware はリクエストとレスポンスをログ出力します
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// リクエスト開始時刻
		startTime := time.Now()

		// リクエスト情報のログ
		log.Printf("========================================")
		log.Printf("[REQUEST] %s %s", c.Request.Method, c.Request.URL.Path)
		log.Printf("From: %s", c.ClientIP())
		
		// クエリパラメータがある場合
		if c.Request.URL.RawQuery != "" {
			log.Printf("Query: %s", c.Request.URL.RawQuery)
		}

		// POSTリクエストでJSONボディがある場合
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			// Content-Typeをチェック
			contentType := c.GetHeader("Content-Type")
			log.Printf("Content-Type: %s", contentType)

			// JSONの場合のみボディを読み取る
			if contentType == "application/json" {
				// ボディを読み取る
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err == nil && len(bodyBytes) > 0 {
					// ボディを再度読めるようにする
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					
					// JSONをパースしてフォーマット
					var jsonData interface{}
					if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
						prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
						log.Printf("Request Body:\n%s", string(prettyJSON))
					} else {
						// JSONパースに失敗した場合は生データを表示
						log.Printf("Request Body (raw): %s", string(bodyBytes))
					}
				}
			} else if contentType == "multipart/form-data" || contentType == "application/x-www-form-urlencoded" {
				// フォームデータの場合
				c.Request.ParseMultipartForm(32 << 20) // 32MB
				if c.Request.MultipartForm != nil {
					log.Printf("Form Values: %v", c.Request.MultipartForm.Value)
					for key, files := range c.Request.MultipartForm.File {
						for _, file := range files {
							log.Printf("File uploaded - Field: %s, Filename: %s, Size: %d bytes", 
								key, file.Filename, file.Size)
						}
					}
				} else if c.Request.Form != nil {
					log.Printf("Form Data: %v", c.Request.Form)
				}
			}
		}

		// レスポンスライターをラップ
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 次のハンドラーを実行
		c.Next()

		// レスポンス情報のログ
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()
		
		log.Printf("[RESPONSE] Status: %d", statusCode)
		log.Printf("Duration: %v", duration)

		// レスポンスボディをログ
		responseContentType := c.Writer.Header().Get("Content-Type")
		if responseContentType != "" {
			log.Printf("Response Content-Type: %s", responseContentType)
		}
		if blw.body.Len() > 0 {
			// Content-Typeのチェック（application/jsonで始まるかチェック）
			if responseContentType == "" || responseContentType == "application/json" || 
			   len(responseContentType) >= 16 && responseContentType[:16] == "application/json" {
				var jsonData interface{}
				if err := json.Unmarshal(blw.body.Bytes(), &jsonData); err == nil {
					// レスポンスサイズが大きすぎる場合は省略
					if blw.body.Len() > 10000 {
						log.Printf("Response Body: (size: %d bytes, content omitted due to size)", blw.body.Len())
					} else {
						prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
						log.Printf("Response Body:\n%s", string(prettyJSON))
					}
				} else {
					// JSONパースに失敗した場合
					log.Printf("Response Body (raw): %s", blw.body.String())
				}
			} else {
				// JSON以外のレスポンス
				if blw.body.Len() > 1000 {
					log.Printf("Response Body: (non-JSON, size: %d bytes)", blw.body.Len())
				} else {
					log.Printf("Response Body: %s", blw.body.String())
				}
			}
		}

		// エラーがある場合
		if len(c.Errors) > 0 {
			log.Printf("Errors: %v", c.Errors.String())
		}

		log.Printf("========================================")
	}
}

// SimpleLoggingMiddleware はシンプルなログ出力（アクセスログのみ）
func SimpleLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// カスタムログフォーマット
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}