# Structure Steering

## ディレクトリ構成

```
sample-api/
├── app/
│   └── main.go              # DI 配線とサーバー起動
├── db/
│   ├── migrate/             # DB schema migration (golang-migrate, .up.sql のみ)
│   └── seed/                # 初期データ・開発用データ (DML のみ)
├── domain/                  # コアドメイン層（フレームワーク依存ゼロ）
│   ├── *.go                 # ドメインモデル（struct + json タグ）。group.go（Group / GroupRelation / GroupMember / SourceGroup を含む）、user.go
│   └── errors.go            # センチネルエラーの一元管理
├── auth/                    # 認証ユースケース層
│   ├── service.go           # auth.Service（GetByUUID）+ UserRepository インターフェース
│   ├── service_test.go      # 外部テストパッケージ (package auth_test)
│   └── mocks/               # テスト用 mock（手動保守）
│       └── user_repository_mock.go
├── {feature}/               # ユースケース層（機能ごとにパッケージを作成）
│   ├── service.go           # Service struct + コンストラクタ + メソッド
│   ├── service_test.go      # 外部テストパッケージ (package {feature}_test)
│   └── mocks/               # テスト用 mock（手動保守）
│       ├── {feature}_repository_mock.go
│       ├── group_relation_repository_mock.go  # group.GroupRelationRepository の mock（group/mocks/ 配下に配置）
│       └── user_repository_mock.go  # group.UserRepository（CountByIDs）の mock（group/mocks/ 配下に配置）
├── internal/
│   ├── repository/
│   │   └── mysql/           # Repository adapter（MySQL 実装）
│   │       ├── {feature}.go          # group.go, group_relation.go（GroupRelationRepository 実装）, user.go
│   │       └── {feature}_test.go
│   └── rest/                # Delivery 層（Echo ハンドラ）
│       ├── {feature}.go       # ハンドラ + インターフェース定義 + ルート登録
│       ├── {feature}_test.go  # モックを使ったハンドラテスト
│       ├── errors.go          # エラー → HTTP ステータスコードのマッピング
│       ├── params.go          # クエリ・パスパラメータの共通パース（parseLimit / parseOffset / parsePathID）
│       ├── health.go          # DBPinger インターフェース定義 + /health ハンドラ登録
│       ├── access_log.go        # AccessLogMiddleware（認証済みリクエストの構造化ログ出力）
│       ├── access_log_test.go   # AccessLogMiddleware のテスト
│       ├── auth.go              # AuthHandler + AuthMiddleware + AuthService インターフェース
│       ├── auth_test.go         # AuthMiddleware・AuthHandler のテスト
│       └── mocks/             # テスト用 mock（手動保守）
│           ├── {feature}_service_mock.go
│           └── auth_service_mock.go  # MockAuthService（rest.AuthService インターフェースのモック）
├── .env.local               # 環境変数（ローカル用、git ignore）
├── .env.local.example       # ローカル環境変数のサンプル
├── .env.docker.example      # Docker 用環境変数のサンプル
├── .golangci.yml            # golangci-lint 設定
├── bin/                     # ビルド成果物
├── docker-compose.yml       # ローカル開発用 MySQL コンテナ定義
├── entrypoint.sh            # Docker API 起動前の migration / seed
├── Makefile                 # ビルド・テスト・lint コマンド
└── README.md                # プロジェクト説明
```

例: `group/`, `user/`, `auth/`

## 命名パターン

| 要素 | パターン | 例 |
|------|---------|-----|
| ユースケースパッケージ | 機能名（小文字） | `group` |
| Service 型 | `Service` | `group.Service` |
| コンストラクタ | `New{Type}` | `group.NewService(repo)` |
| ハンドラ型 | `{Feature}Handler` | `rest.GroupHandler` |
| ハンドラ登録関数 | `New{Feature}Handler` | `rest.NewGroupHandler(e, svc)` |
| Service IF（rest 側） | `{Feature}Service` | `rest.GroupService` |
| Repository IF（feature 側） | `{Feature}Repository` | `group.GroupRepository` |
| Repository 実装 | `{Feature}Repository` | `mysql.GroupRepository` |
| テスト用 mock 型 | `Mock{Feature}{IF}` | `mocks.MockGroupRepository`, `mocks.MockGroupService` |

## DI パターン（app/main.go）

```go
e := echo.New()
e.Use(middleware.CORS())           // ミドルウェア登録

// group: repository → service → handler の標準パターン
// GroupService は GroupRepository / UserRepository / GroupRelationRepository の 3 つを受け取る
groupRepo := mysql.NewGroupRepository(db)
userRepo := mysql.NewUserRepository(db)
groupRelationRepo := mysql.NewGroupRelationRepository(db)
gSvc := group.NewServiceWithRelation(groupRepo, userRepo, groupRelationRepo)

// user: user 一覧の標準パターン
uSvc := user.NewService(userRepo)

// auth: 認証ミドルウェア + me エンドポイント
// /api/v1 以下のルートグループに AuthMiddleware を適用する
apiGroup := e.Group("/api/v1")
aSvc := auth.NewService(userRepo)  // userRepo を共有
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
apiGroup.Use(rest.AuthMiddleware(appEnv, aSvc))
apiGroup.Use(rest.AccessLogMiddleware(logger))  // AuthMiddleware の後に登録（authUser が設定済みの状態でログ）
rest.NewAuthHandler(apiGroup)  // AuthHandler は AuthService を保持しない
rest.NewGroupHandler(apiGroup, gSvc)
rest.NewUserHandler(apiGroup, uSvc)
```

新しいドメインを追加する場合は group パターン（Repository → Service → Handler）を踏襲する。`GroupRelationRepository` のような補助 repository が増えた場合は `NewXxxWithRelation` パターンで追加する。

> **補足**: `mysql.UserRepository` は `group.UserRepository`、`user.UserRepository`、`auth.UserRepository` の 3 つのインターフェースを実装している。複数のサービスから共有されるリポジトリ実装は 1 つのインスタンスを共有して DI する。

> **handler メソッドと service IF の対応**: handler のメソッド名は必ずしも service IF と一致しない場合がある。例として `GroupHandler.DeleteGroupMembers` は `GroupService.RemoveGroupMembers` を呼ぶ。外部 HTTP 動詞（DELETE）と内部ユースケース名（Remove）の意味論的差異が命名に反映されている。

## 新規ドメイン追加時の手順

1. `domain/{model}.go` にドメインモデルを定義
2. `{feature}/service.go` にユースケースを実装
3. `internal/repository/mysql/{feature}.go` に Repository adapter を実装
4. `internal/rest/{feature}.go` にハンドラとインターフェースを定義
5. `app/main.go` で DI 配線とルート登録を追加

詳細は `CLAUDE.md` の `/go-clean-arch` スキルを参照。
