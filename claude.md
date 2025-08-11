# 電帳君 APIサーバー開発仕様書

## プロジェクト概要

**電帳君（Denchokun）** は電子帳簿保存法対応のデジタル帳簿管理システムです。現在SQLiteでローカル管理しているデータを、REST APIサーバー経由でアクセスするように変更します。

## システム要件

### 技術スタック
- **データベース**: SQLite（既存DBとの互換性を保つ）
- **API**: REST API（JSON形式）
- **プロトコル**: HTTP/HTTPS
- **ポート**: 8080（デフォルト）

### 対象クライアント
- **OS**: Windows 64bit
- **言語**: Xojo
- **通信**: HTTPSocket使用

## データベーススキーマ

### 1. Deals テーブル（取引データ）
```sql
CREATE TABLE "Deals" (
    "NO" TEXT NOT NULL UNIQUE,          -- 取引番号（プライマリキー）
    "nextNO" TEXT,                      -- 次の取引番号（リンク）
    "prevNO" TEXT,                      -- 前の取引番号（リンク）
    "DealType" TEXT,                    -- 取引種別（領収書、請求書等）
    "DealDate" TEXT,                    -- 取引日（YYYY-MM-DD形式）
    "DealName" TEXT,                    -- 取引名
    "DealPartner" TEXT,                 -- 取引相手
    "DealPrice" INTEGER,                -- 取引金額
    "DealRemark" TEXT,                  -- 備考
    "RecUpdate" TEXT,                   -- 更新日時
    "RegDate" TEXT,                     -- 登録日時
    "RecStatus" TEXT,                   -- レコード状態（NEW/UPDATE）
    "FilePath" TEXT,                    -- ファイルパス
    "Hash" TEXT,                        -- ファイルハッシュ値
    PRIMARY KEY("NO")
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_Hash ON Deals (Hash);
```

### 2. System テーブル（システム情報）
```sql
CREATE TABLE "System" (
    "AppVersion" TEXT,                  -- アプリバージョン
    "SQLiteLibraryVersion" TEXT         -- SQLiteライブラリバージョン
);
```

### 3. DealPartners テーブル（取引先マスタ）
```sql
CREATE TABLE DealPartners (
    name TEXT PRIMARY KEY               -- 取引先名
);
```

## API エンドポイント仕様

### ベースURL
```
http://localhost:8080/api/v1
```

### 1. ヘルスチェック

#### `GET /health`
**説明**: サーバーの稼働状況確認

**レスポンス**:
```json
{
    "status": "ok",
    "message": "Server is running",
    "timestamp": "2024-01-15T10:30:00Z"
}
```

### 2. 期間管理

#### `GET /periods`
**説明**: 利用可能な取引期間一覧を取得

**レスポンス**:
```json
{
    "success": true,
    "periods": ["2024-01", "2024-02", "2024-03", "2024-04"]
}
```

#### `POST /periods/{period}/connect`
**説明**: 指定期間のデータベース接続を確立

**パラメータ**:
- `period`: 期間名（例: "2024-01"）

**レスポンス**:
```json
{
    "success": true,
    "message": "Connected to period 2024-01",
    "databasePath": "/data/2024-01/Denchokun.db"
}
```

### 3. 取引データ管理

#### `POST /deals`
**説明**: 新しい取引データを登録

**リクエストボディ**:
```json
{
    "period": "2024-01",
    "dealData": {
        "NO": "D240115001",
        "DealType": "領収書",
        "DealDate": "2024-01-15",
        "DealName": "文房具購入",
        "DealPartner": "オフィス用品店",
        "DealPrice": 1500,
        "DealRemark": "ペン、ノート等",
        "RecStatus": "NEW"
    },
    "fileData": {
        "name": "receipt_20240115.pdf",
        "path": "D240115001_2024-01-15_オフィス用品店_1500.pdf",
        "size": 245760,
        "hash": "abc123def456...",
        "base64Data": "JVBERi0xLjQKJdPr6eEKMSAwIG9iago8PAo..."  // ファイルが10MB未満の場合
    }
}
```

**レスポンス（成功）**:
```json
{
    "success": true,
    "message": "Deal created successfully",
    "dealId": "D240115001",
    "filePath": "D240115001_2024-01-15_オフィス用品店_1500.pdf"
}
```

**レスポンス（エラー）**:
```json
{
    "success": false,
    "error": "Database error: UNIQUE constraint failed",
    "message": "Deal number already exists"
}
```

#### `GET /deals`
**説明**: 取引データの検索・取得

**クエリパラメータ**:
- `period`: 期間（必須）
- `from_date`: 開始日（YYYY-MM-DD）
- `to_date`: 終了日（YYYY-MM-DD）
- `partner`: 取引先名（部分一致）
- `type`: 取引種別
- `keyword`: キーワード検索

**例**: `/deals?period=2024-01&from_date=2024-01-01&to_date=2024-01-31`

**レスポンス**:
```json
{
    "success": true,
    "count": 150,
    "deals": [
        {
            "NO": "D240115001",
            "DealType": "領収書",
            "DealDate": "2024-01-15",
            "DealName": "文房具購入",
            "DealPartner": "オフィス用品店",
            "DealPrice": 1500,
            "DealRemark": "ペン、ノート等",
            "RecUpdate": "2024-01-15T10:30:00Z",
            "RegDate": "2024-01-15T10:30:00Z",
            "RecStatus": "NEW",
            "FilePath": "D240115001_2024-01-15_オフィス用品店_1500.pdf",
            "Hash": "abc123def456..."
        }
    ]
}
```

#### `PUT /deals/{dealId}`
**説明**: 取引データの更新

**パラメータ**:
- `dealId`: 取引番号

**リクエストボディ**: `POST /deals` と同様

#### `DELETE /deals/{dealId}`
**説明**: 取引データの削除

**パラメータ**:
- `dealId`: 取引番号

**レスポンス**:
```json
{
    "success": true,
    "message": "Deal deleted successfully"
}
```

### 4. ファイル管理

#### `POST /files`
**説明**: ファイルのアップロード（大容量ファイル用）

**リクエスト**: `multipart/form-data`
- `file`: アップロードファイル
- `period`: 期間名
- `dealId`: 取引番号

#### `GET /files/{fileId}`
**説明**: ファイルのダウンロード

### 5. マスタデータ管理

#### `GET /deal-partners`
**説明**: 取引先マスタの取得

**レスポンス**:
```json
{
    "success": true,
    "partners": ["株式会社A", "B商店", "C工業"]
}
```

#### `POST /deal-partners`
**説明**: 取引先の追加

**リクエストボディ**:
```json
{
    "name": "新しい取引先"
}
```

## 同時アクセス対応

### 複数クライアント要件
- **想定クライアント数**: 5-10台の同時接続
- **主な操作**: 取引データの登録・検索・更新
- **競合の可能性**: 同じ期間への同時書き込み

### 排他制御方針

#### 1. データベースレベル
```sql
-- トランザクション分離レベル
PRAGMA journal_mode = WAL;          -- Write-Ahead Logging
PRAGMA synchronous = NORMAL;        -- パフォーマンスと安全性のバランス
PRAGMA busy_timeout = 30000;        -- 30秒のタイムアウト
```

#### 2. アプリケーションレベル
- **楽観的ロック**: 更新時にバージョンチェック
- **悲観的ロック**: 重要な操作時の明示的ロック
- **取引番号生成**: 重複防止のため UUID または タイムスタンプベース

#### 3. ファイル操作
- **原子性**: ファイル書き込み時の一時ファイル使用
- **重複チェック**: ハッシュ値による同一ファイル検出
- **ロック**: ファイル操作中の排他制御

### 具体的な実装例

#### 取引データ登録時の排他制御
```go
// 1. 取引番号の重複チェック
func (s *Server) createDeal(c *gin.Context) {
    tx, err := s.db.Begin()
    if err != nil {
        c.JSON(500, gin.H{"error": "Transaction start failed"})
        return
    }
    defer tx.Rollback()
    
    // 2. 重複チェック（SELECT FOR UPDATE）
    var exists int
    err = tx.QueryRow("SELECT COUNT(*) FROM Deals WHERE NO = ? FOR UPDATE", dealNo).Scan(&exists)
    if exists > 0 {
        c.JSON(409, gin.H{"error": "Deal number already exists"})
        return
    }
    
    // 3. データ挿入
    _, err = tx.Exec("INSERT INTO Deals (...) VALUES (...)")
    if err != nil {
        c.JSON(500, gin.H{"error": "Insert failed"})
        return
    }
    
    // 4. コミット
    if err = tx.Commit(); err != nil {
        c.JSON(500, gin.H{"error": "Commit failed"})
        return
    }
    
    c.JSON(200, gin.H{"success": true})
}
```

#### ファイル保存時の競合対策
```go
func saveFileAtomic(filePath string, data []byte) error {
    // 1. 一時ファイルに書き込み
    tempFile := filePath + ".tmp"
    if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
        return err
    }
    
    // 2. 原子的にリネーム
    return os.Rename(tempFile, filePath)
}
```

### 同時アクセス時のエラーハンドリング

#### HTTPステータスコード
- `409 Conflict`: リソースの競合（取引番号重複等）
- `423 Locked`: リソースがロック中
- `429 Too Many Requests`: 同時アクセス数制限

#### エラーレスポンス例
```json
{
    "success": false,
    "error": "resource_conflict",
    "message": "Deal number D240115001 already exists",
    "retryAfter": 1000,
    "suggestions": [
        "Use auto-generated deal number",
        "Check existing deals first"
    ]
}
```

### パフォーマンス最適化

#### 1. 接続プール
```go
// データベース接続プール設定
db.SetMaxOpenConns(25)    // 最大同時接続数
db.SetMaxIdleConns(5)     // アイドル接続数
db.SetConnMaxLifetime(time.Hour)  // 接続の最大寿命
```

#### 2. インデックス最適化
```sql
-- 検索性能向上のためのインデックス
CREATE INDEX idx_deal_date ON Deals(DealDate);
CREATE INDEX idx_deal_partner ON Deals(DealPartner);
CREATE INDEX idx_deal_type ON Deals(DealType);
CREATE INDEX idx_hash ON Deals(Hash);
```

#### 3. キャッシュ戦略
- **期間一覧**: メモリキャッシュ（10分間）
- **取引先マスタ**: メモリキャッシュ（更新時に無効化）
- **ファイルメタデータ**: Redis等の外部キャッシュ検討

## セキュリティ要件

### 認証
- 初期版では認証なし（ローカルネットワーク前提）
- 将来的にはAPIキー認証を検討

### データ検証
- SQLインジェクション対策必須
- ファイルサイズ制限（最大100MB）
- 許可するファイル形式の制限

## エラーハンドリング

### HTTPステータスコード
- `200`: 成功
- `400`: リクエストエラー
- `404`: リソースが見つからない
- `500`: サーバーエラー

### エラーレスポンス形式
```json
{
    "success": false,
    "error": "error_code",
    "message": "Human readable error message",
    "details": {
        "field": "validation error details"
    }
}
```

## ファイル管理要件

### ファイル保存パス
```
{base_path}/{period}/{dealId}__{date}__{partner}__{price}.{ext}
```

例: `D240115001_2024-01-15_オフィス用品店_1500.pdf`

### フォルダ構造
```
/data/
├── 2024-01/
│   ├── Denchokun.db
│   ├── D240115001_2024-01-15_オフィス用品店_1500.pdf
│   └── ...
├── 2024-02/
│   └── ...
```

### ハッシュ値計算
- **アルゴリズム**: SHA-256
- **用途**: 重複ファイルチェック、整合性確認

## パフォーマンス要件

### レスポンス時間
- 単一取引取得: 100ms以内
- 検索（100件）: 500ms以内
- ファイルアップロード: ファイルサイズに依存

### 同時接続
- **目標**: 5-10台のクライアント同時接続
- **最大**: 25接続（データベース接続プール設定）
- **レート制限**: クライアントあたり 100リクエスト/分
- **タイムアウト**: 30秒（データベースビジー時）

## 開発・テスト環境

### 推奨開発環境
- **Go**: 1.19+ with Gin/Echo framework（推奨）
- SQLiteドライバー: github.com/mattn/go-sqlite3
- その他必要パッケージ:
  - github.com/gin-gonic/gin（Webフレームワーク）
  - github.com/google/uuid（UUID生成）
  - crypto/sha256（ハッシュ計算）

### Go言語実装例

```go
// main.go - 基本的なAPIサーバー構造
package main

import (
    "database/sql"
    "net/http"
    
    "github.com/gin-gonic/gin"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    r := gin.Default()
    
    // ヘルスチェック
    r.GET("/api/v1/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "ok",
            "message": "Server is running",
        })
    })
    
    // 期間一覧取得
    r.GET("/api/v1/periods", getPeriods)
    
    // 取引データ管理
    r.POST("/api/v1/deals", createDeal)
    r.GET("/api/v1/deals", getDeals)
    r.PUT("/api/v1/deals/:id", updateDeal)
    r.DELETE("/api/v1/deals/:id", deleteDeal)
    
    r.Run(":8080")
}
```

### 必要なGo modules
```bash
go mod init denchokun-api
go get github.com/gin-gonic/gin
go get github.com/mattn/go-sqlite3
go get github.com/google/uuid
```

### テストデータ
テスト用のサンプルデータを含むSQLiteファイルを提供予定

## API開発優先順位

アプリケーション側の開発・テスト効率を考慮した段階的な開発順序を以下に示します。

### フェーズ1: 基盤API（最優先）

#### 1.1 サーバー稼働確認
```
GET /api/v1/health
```
- **理由**: サーバーとの基本的な通信確認
- **テスト**: APIClientClass.TestConnection() の実装
- **所要時間**: 1-2時間

#### 1.2 期間管理
```
GET /api/v1/periods
POST /api/v1/periods/{period}/connect
```
- **理由**: メイン画面の期間プルダウン表示に必要
- **テスト**: ReCreateDealPeriodPopupMenu() の動作確認
- **所要時間**: 4-6時間

### フェーズ2: 取引データ登録（高優先）

#### 2.1 取引データ登録
```
POST /api/v1/deals
```
- **理由**: アプリの主要機能、ファイルアップロード含む
- **テスト**: MainWindow の登録ボタン動作確認
- **所要時間**: 8-12時間
- **注意**: ファイルハンドリング、ハッシュ計算、重複チェック含む

#### 2.2 取引データ検索・取得
```
GET /api/v1/deals
```
- **理由**: 登録したデータの表示・確認
- **テスト**: DealPeriodWindow の一覧表示
- **所要時間**: 6-8時間

### フェーズ3: データ管理機能（中優先）

#### 3.1 取引先マスタ管理
```
GET /api/v1/deal-partners
POST /api/v1/deal-partners
PUT /api/v1/deal-partners/{id}
DELETE /api/v1/deal-partners/{id}
```
- **理由**: 取引先入力の利便性向上
- **テスト**: DealPartnersMasterWindow の動作確認
- **所要時間**: 4-6時間

#### 3.2 取引データ更新・削除
```
PUT /api/v1/deals/{dealId}
DELETE /api/v1/deals/{dealId}
```
- **理由**: データメンテナンス機能
- **テスト**: 既存データの修正・削除操作
- **所要時間**: 4-6時間

### フェーズ4: ファイル管理（中優先）

#### 4.1 ファイルダウンロード
```
GET /api/v1/files/{fileId}
```
- **理由**: 登録済みファイルの表示・確認
- **テスト**: DetailWindow でのファイル表示
- **所要時間**: 3-4時間

#### 4.2 大容量ファイルアップロード
```
POST /api/v1/files
```
- **理由**: 10MB超のファイル対応
- **テスト**: 大きなPDFファイルの登録
- **所要時間**: 4-6時間

### フェーズ5: 検索・フィルタ機能（低優先）

#### 5.1 高度な検索機能
```
GET /api/v1/deals?keyword=XXX&date_range=XXX
```
- **理由**: 業務効率化
- **テスト**: 複雑な検索条件での絞り込み
- **所要時間**: 6-8時間

### 開発のマイルストーン

#### 🎯 マイルストーン1（フェーズ1完了）
- **目標**: アプリ起動とサーバー通信確認
- **確認事項**: 
  - MainWindow が正常表示
  - 期間プルダウンに値が表示
  - サーバー接続エラーがない

#### 🎯 マイルストーン2（フェーズ2完了）
- **目標**: 基本的な取引登録・表示
- **確認事項**:
  - ファイルドロップで取引登録可能
  - 登録したデータが検索で表示
  - エラーハンドリングが適切

#### 🎯 マイルストーン3（フェーズ3完了）
- **目標**: 完全なCRUD操作
- **確認事項**:
  - データの追加・更新・削除
  - 取引先マスタ管理
  - 複数クライアントでの動作確認

### 各フェーズでのテスト項目

#### フェーズ1テスト
```bash
# ヘルスチェック
curl http://localhost:8080/api/v1/health

# 期間一覧
curl http://localhost:8080/api/v1/periods
```

#### フェーズ2テスト
```bash
# 取引登録（サンプルデータ）
curl -X POST http://localhost:8080/api/v1/deals \
  -H "Content-Type: application/json" \
  -d '{"period":"2024-01","dealData":{...}}'

# 取引検索
curl "http://localhost:8080/api/v1/deals?period=2024-01"
```

### 優先順位の理由

1. **フェーズ1**: アプリの基本動作に必須
2. **フェーズ2**: メイン機能、ユーザーが最も使用
3. **フェーズ3**: データ管理、業務効率向上
4. **フェーズ4**: 大容量対応、パフォーマンス
5. **フェーズ5**: 付加価値機能

### 並行開発可能な箇所
- **フェーズ1** と **フェーズ3.1**（取引先マスタ）は並行開発可能
- **フェーズ2.1** 完了後、**フェーズ2.2** と **フェーズ3.2** は並行開発可能

**総開発時間見積もり**: 40-60時間（1人で約1-2週間）

## 今後の拡張予定

### フェーズ2
- 認証機能
- ログ機能
- バックアップ・復元機能

### フェーズ3
- 複数データベースサポート
- 分散処理対応
- リアルタイム同期

---

## 開発者への注意事項

1. **既存データ互換性**: 既存のSQLiteデータベースと完全互換性を保つこと
2. **エラーハンドリング**: データベースエラーは適切にHTTPステータスコードに変換
3. **ファイル操作**: ファイルのコピー・移動・削除処理を含む
4. **文字エンコーディング**: UTF-8対応必須（日本語データ）
5. **ログ出力**: 十分なデバッグ情報を含むログ

このAPIサーバーは電帳君アプリケーションの核となる部分です。安定性とパフォーマンスを重視した実装をお願いいたします。


---

## 期間管理機能の拡張仕様（追加要件）

### 概要
現在の期間管理機能を拡張し、期間名だけでなく各期間の詳細情報（開始日・終了日）もサーバー側で管理する必要があります。複数の電帳君クライアントが同じAPIサーバーを使用するため、期間データの一元管理が必要です。

### データベース拡張

#### 4. Periods テーブル（期間管理）- System.db内
```sql
-- System.db内に作成
CREATE TABLE "Periods" (
    "name" TEXT PRIMARY KEY,           -- 期間名（例: "2024-01"）
    "fromDate" TEXT,                   -- 開始日（例: "2024-01-01" または "未設定"）
    "toDate" TEXT,                     -- 終了日（例: "2024-01-31" または "未設定"）
    "created" TEXT,                    -- 作成日時（ISO 8601形式）
    "updated" TEXT                     -- 更新日時（ISO 8601形式）
);
```

#### 初期データ
```sql
INSERT INTO Periods VALUES 
('2024-01', '2024-01-01', '2024-01-31', datetime('now'), datetime('now'));
```

### API拡張要件

#### 既存APIの拡張: `GET /periods`

**現在のレスポンス:**
```json
{
    "success": true,
    "periods": ["2024-01", "2024-02", "2024-03"]
}
```

**拡張後のレスポンス:**
```json
{
    "success": true,
    "periods": [
        {
            "name": "2024-01",
            "fromDate": "2024-01-01",
            "toDate": "2024-01-31"
        },
        {
            "name": "2024-02",
            "fromDate": "2024-02-01",
            "toDate": "2024-02-29"
        },
        {
            "name": "2024-03",
            "fromDate": "未設定",
            "toDate": "未設定"
        }
    ]
}
```

#### 新規API エンドポイント

#### `GET /periods/{period}`
**説明**: 指定期間の詳細情報を取得

**パラメータ**:
- `period`: 期間名（例: "2024-01"）

**レスポンス**:
```json
{
    "success": true,
    "period": {
        "name": "2024-01",
        "fromDate": "2024-01-01",
        "toDate": "2024-01-31",
        "created": "2024-01-15T10:30:00Z",
        "updated": "2024-01-15T10:30:00Z"
    }
}
```

#### `POST /periods`
**説明**: 新しい期間を作成

**リクエストボディ**:
```json
{
    "name": "2024-05",
    "fromDate": "2024-05-01",
    "toDate": "2024-05-31"
}
```

**レスポンス**:
```json
{
    "success": true,
    "message": "Period created successfully",
    "period": {
        "name": "2024-05",
        "fromDate": "2024-05-01",
        "toDate": "2024-05-31",
        "created": "2024-01-15T10:30:00Z",
        "updated": "2024-01-15T10:30:00Z"
    }
}
```

#### `PUT /periods/{period}`
**説明**: 既存期間の詳細情報を更新

**パラメータ**:
- `period`: 期間名（例: "2024-01"）

**リクエストボディ**:
```json
{
    "fromDate": "2024-01-01",
    "toDate": "2024-01-31"
}
```

**レスポンス**:
```json
{
    "success": true,
    "message": "Period updated successfully",
    "period": {
        "name": "2024-01",
        "fromDate": "2024-01-01",
        "toDate": "2024-01-31",
        "created": "2024-01-15T10:30:00Z",
        "updated": "2024-01-15T12:45:00Z"
    }
}
```

#### `DELETE /periods/{period}`
**説明**: 期間を削除

**パラメータ**:
- `period`: 期間名（例: "2024-01"）

**レスポンス**:
```json
{
    "success": true,
    "message": "Period deleted successfully"
}
```

**エラーレスポンス（取引データが存在する場合）**:
```json
{
    "success": false,
    "error": "period_has_deals",
    "message": "Cannot delete period that contains deal records"
}
```

### バリデーション要件

#### 期間作成・更新時のバリデーション
1. **期間名の形式**: `YYYY-MM` 形式を推奨（例: "2024-01"）
2. **日付形式**: `YYYY-MM-DD` 形式または `"未設定"`
3. **日付の妥当性**: fromDate ≤ toDate（両方とも設定されている場合）
4. **重複チェック**: 同名期間の重複作成を防止

#### 期間削除時のチェック
- 削除対象期間に取引データが存在する場合は削除を拒否
- 適切なエラーメッセージを返す

### 実装優先順位
1. **Periodsテーブルの作成とマイグレーション**
2. **GET /periods の拡張**（既存クライアントが使用中）
3. **GET /periods/{period} の実装**
4. **POST/PUT/DELETE の実装**

### 互換性注意事項
- 既存の `GET /periods` エンドポイントのレスポンス形式変更
- クライアント側でのレスポンス形式変更への対応が必要
- 段階的な移行を推奨

### データ移行
既存の期間データがある場合：
1. 既存の期間名をPeriodsテーブルに移行
2. fromDate, toDateは初期値として "未設定" を設定
3. 後から管理画面で設定可能