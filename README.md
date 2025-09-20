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

### 環境変数による設定

サーバーの設定は以下の環境変数で行います：

| 環境変数 | 説明 | デフォルト値 |
|----------|------|-------------|
| `DENCHOKUN_BASEPATH` | データベースファイルの保存先（絶対パス） | `./data` |
| `DENCHOKUN_PORT` | サーバーのポート番号 | `:8080` |
| `DENCHOKUN_MODE` | 実行モード（debug/release） | `debug` |

#### Windows での設定例
```batch
set DENCHOKUN_BASEPATH=C:\Users\motoi\DenchokunData
set DENCHOKUN_PORT=:9000
set DENCHOKUN_MODE=release
denchokun-api.exe
```

#### Linux/macOS での設定例
```bash
export DENCHOKUN_BASEPATH=/home/user/denchokun-data
export DENCHOKUN_PORT=:9000
export DENCHOKUN_MODE=release
./denchokun-api
```

#### 複数のサーバーで同じデータディレクトリを使用する場合
```batch
REM プレビューサーバーと同じデータを共有
set DENCHOKUN_BASEPATH=C:\SharedData\Denchokun
set DENCHOKUN_PORT=:8080
start denchokun-api.exe

REM 別のポートで2つ目のサーバーを起動
set DENCHOKUN_PORT=:8081
start denchokun-api.exe
```

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
