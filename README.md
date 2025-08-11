# 電帳君 APIサーバー

電子帳簿保存法対応のデジタル帳簿管理システム「電帳君（Denchokun）」のREST APIサーバーです。

## 概要

このサーバーは、既存のSQLiteデータベースベースの電帳君アプリケーションを複数クライアントから同時アクセス可能にするためのAPIサーバーです。

## 機能

- **期間管理**: 月別でデータを管理
- **取引データ管理**: CRUD操作対応
- **ファイル管理**: 領収書等のファイルアップロード/ダウンロード
- **取引先マスタ管理**: 取引先情報の管理
- **同時アクセス対応**: 複数クライアントからの同時接続サポート

## セットアップ

### 必要要件

- Go 1.19以上
- SQLite3

### インストール

```bash
# 依存関係のダウンロード
go mod download

# ビルド
go build -o denchokun-server

# 実行
./denchokun-server
```

### Windows環境での実行

```powershell
# ビルド
go build -o denchokun-server.exe

# 実行
.\denchokun-server.exe
```

## 設定

`config.json`ファイルで以下の設定が可能です：

```json
{
  "server": {
    "port": ":8080",
    "mode": "debug"
  },
  "database": {
    "basePath": "./data"
  }
}
```

- `port`: サーバーのポート番号
- `mode`: 実行モード（"debug" または "release"）
- `basePath`: データベースファイルの保存先ディレクトリ

## API エンドポイント

### ベースURL
```
http://localhost:8080/api/v1
```

### 主要エンドポイント

#### ヘルスチェック
```
GET /health
```

#### 期間管理
```
GET /periods                     # 利用可能な期間一覧
POST /periods/:period/connect    # 指定期間への接続
```

#### 取引データ
```
GET /deals                       # 取引データ検索
POST /deals                      # 新規取引登録
GET /deals/:dealId               # 取引データ取得
PUT /deals/:dealId               # 取引データ更新
DELETE /deals/:dealId            # 取引データ削除
```

#### ファイル管理
```
POST /files                      # ファイルアップロード
GET /files/:fileId               # ファイルダウンロード
```

#### 取引先マスタ
```
GET /deal-partners               # 取引先一覧
POST /deal-partners              # 取引先登録
PUT /deal-partners/:name         # 取引先更新
DELETE /deal-partners/:name      # 取引先削除
```

## テスト

### Windows PowerShell
```powershell
.\scripts\test_api.ps1
```

### Linux/Mac
```bash
./scripts/test_api.sh
```

## データベース構造

データは期間（月）ごとに独立したSQLiteデータベースファイルとして管理されます：

```
data/
├── 2024-01/
│   ├── Denchokun.db
│   └── [添付ファイル]
├── 2024-02/
│   ├── Denchokun.db
│   └── [添付ファイル]
└── ...
```

## 同時アクセス対応

- WAL（Write-Ahead Logging）モードで動作
- 最大25接続まで対応
- トランザクション制御による排他制御実装

## ライセンス

商用ソフトウェア

## サポート

問題が発生した場合は、開発者までお問い合わせください。
