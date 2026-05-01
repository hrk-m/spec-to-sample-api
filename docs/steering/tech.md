# Tech Steering

## スタック

- **言語**: Go 1.25
- **HTTP フレームワーク**: Echo v4 (`labstack/echo`)
- **ミドルウェア**: CORS（`echo/v4/middleware`）
- **DB ドライバ**: `go-sql-driver/mysql`
- **テスト**: testify (`assert` + `mock`)
- **Lint**: golangci-lint v2
- **インフラ**: Docker Compose（MySQL）

## アーキテクチャ決定

### Clean Architecture の採用

3 層に明確に分離し、依存方向を内側（domain）に向ける。

```
internal/rest/              →  {feature}/           →  domain/
  (delivery)                   (use case)              (entity)
internal/repository/mysql/  →  {feature} repository IF
  (repository adapter)
db/migrate/                 →  DB schema migration (golang-migrate)
db/seed/                    →  Seed data (DML only)
```

- `domain/`: フレームワーク依存ゼロ。純粋な struct とセンチネルエラーのみ
- `internal/rest/health.go`: `DBPinger` インターフェースを消費側（`internal/rest/`）で定義し、`*sql.DB` が実装する。健全性チェックは `RegisterHealthHandler(e, db)` で `GET /health` に登録する
- `group/`、`user/`、`auth/` など機能別パッケージ: ビジネスロジックを実装し、repository interface を宣言する
- `internal/repository/mysql/`: MySQL ベースの repository adapter を実装する
- `internal/rest/`: Echo ハンドラ。上位層のインターフェースを定義し、DI で受け取る

### インターフェース定義の配置

インターフェースは**消費側**で定義する。たとえば `GroupService` は `internal/rest/` が定義し、`GroupRepository` は `group/` が定義する。`UserRepository` も `user/` が定義する。これにより delivery 層と use case 層が実装詳細に依存しない。

### 認証アーキテクチャ

- `auth/service.go` に `auth.Service` を定義。`UserRepository` インターフェース（`GetByUUID`）を消費する
- `internal/rest/auth.go` に `AuthService` インターフェース・`AuthHandler`・`AuthMiddleware` を定義
  - `AuthMiddleware` は `APP_ENV=development` のとき `DEV_USER_UUID` 環境変数から UUID を読み取り、`svc.GetByUUID` でユーザーを取得してコンテキストにセットする
  - `APP_ENV` が `development` 以外の場合は起動時に `log.Fatal` で終了する（本番向け認証は未実装）
- `GET /api/v1/me` はコンテキストから `authUser` を取得して返す（`AuthMiddleware` が事前にセット）。`AuthHandler` は `AuthService` を保持せず、`internal/rest/auth.go` の `NewAuthHandler(g *echo.Group)` はルート登録のみを行う
- `mysql.UserRepository` は `auth.UserRepository`（`GetByUUID`）も実装する

### アクセスログ

- `internal/rest/access_log.go` に `AccessLogMiddleware` を定義。`*slog.Logger` を受け取り、各リクエストを構造化 JSON でログ出力する
  - ログフィールド: `endpoint`（メソッド + パス）、`login_user`（認証ユーザーの UUID）、`latency_s`（秒）、`status`（HTTP ステータスコード）、`header`（Authorization は `[REDACTED]` にマスク）
  - レスポンスヘッダー `X-Login-User` に認証ユーザーの UUID をセットする
  - `AuthMiddleware` の後に登録する必要がある（`authUser` がコンテキストにセット済みの状態でログを取るため）
  - `app/main.go` で `slog.New(slog.NewJSONHandler(os.Stdout, nil))` で生成した logger を渡す

### エラーハンドリング

- `domain/errors.go` にセンチネルエラーを集約
- `internal/rest/errors.go` でエラーを HTTP ステータスコードにマッピング（`ErrBadParamInput` → 400、`ErrNotFound` → 404、`ErrConflict` → 409、`ErrInternalServerError` → 500、その他 → 500）
- ハンドラは `ResponseError{Message}` で JSON エラーレスポンスを返す
- パスパラメータの ID は `internal/rest/params.go` の `parsePathID`（`strconv.ParseUint` ベース）でパースし、変換失敗または `< 1` の場合は `getStatusCode` を通さず直接 400 を返す。同ファイルに `parseLimit`・`parseOffset` も定義されており、クエリパラメータのパースを共通化している

## コーディング規約

- すべての公開シンボルにドキュメントコメントを付ける
- テストファイルは `package xxx_test`（外部テストパッケージ）を使用
- lll: 行の上限 160 文字、funlen: 関数は 150 行・80 文以内
- lint 対象からテストファイルを除外（`.golangci.yml` の `exclude-files`）
- 入力の正規化は service 層で行う: `name = strings.TrimSpace(name)` (Store/Update)、`q = strings.TrimSpace(q)` (ListUsers)。handler 層では正規化しない

## テスト方針

- **use case 層**: repository interface を小さな mock で差し替えてテストする
- **delivery 層**: `testify/mock` で use case をモック化し、httptest でエンドポイントを検証する
- **repository 層**: `//go:build integration` タグ付きの統合テストとして実 DB に接続して検証する（`go test -tags integration ./...` で実行）。`health` ハンドラのテストのみ `go-sqlmock` を使用
- mock は `mocks/` ディレクトリに分離配置する（`{feature}/mocks/` に `MockXxxRepository`、`internal/rest/mocks/` に `MockXxxService`）
- mock は手動保守し、interface 変更時は同じ変更セットで追随させる
- エラー系（センチネルエラー、予期しないエラー）のケースを必ず網羅する

## サービスインターフェース（`GroupService` / `UserService` / `AuthService`）

インターフェースは消費側（`internal/rest/`）で宣言する。

```go
// GroupService: internal/rest/group.go で宣言
type GroupService interface {
    ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error)
    GetByID(ctx context.Context, id uint64) (domain.Group, []domain.Group, error)
    ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.GroupMember, int, error)
    Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error)
    Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error)
    Delete(ctx context.Context, id uint64, userID uint64) error
    ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error)
    AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error)
    RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error
    CreateSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error)
    DeleteSubGroup(ctx context.Context, parentGroupID, childGroupID uint64) error
}

// UserService: internal/rest/user.go で宣言
type UserService interface {
    ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error)
    GetUser(ctx context.Context, id uint64) (*domain.User, error)
}

// AuthService: internal/rest/auth.go で宣言
type AuthService interface {
    GetByUUID(ctx context.Context, uuid string) (domain.User, error)
}
```

`Update` は ID（`uint64`）・name・description・`userID`（操作者の `domain.User.ID`）を受け取り、更新後の `*domain.Group` を返す。`Delete` は ID（`uint64`）と `userID` を受け取り、soft delete を実行する（成功時は `nil`、対象未存在時は `ErrNotFound`）。`userID` は `groups.updated_by` カラムに記録される（`20260417130000_add_updated_by_to_groups.up.sql` で追加）。

`GetByID` は `(domain.Group, []domain.Group, error)` を返す。第 2 戻り値は直属の子グループ一覧（`subgroups`）で、`GroupRelationRepository.ListChildren` から取得する。`relationRepo` が nil の場合（`NewService` で生成した場合）は空スライスを返す。通常は `NewServiceWithRelation` で生成してリレーション機能を有効にする。

`Update` および `Delete` は、`GetByID` や `ListGroupMembers` と同様に、service 層で `id < minID`（`minID = 1`）のバリデーションを行い、不正な ID には `ErrBadParamInput` を返す（repository は呼び出さない）。`userID` は handler 層でコンテキストから取得した `authUser.ID` を渡す。

`ListNonGroupMembers` は `groupID` の存在確認を service 層で行い（`GetByID` 経由）、存在しない場合は `ErrNotFound` を返す。

`AddGroupMembers` は handler 層で `user_ids` の空チェック（`len == 0` → 400）を行い、service 層でまず `deduplicateUint64` によるユーザー ID 重複除去を行い、グループ存在確認と全ユーザー存在確認（`userRepo.CountByIDs` による 1 回の COUNT クエリ）を行い、`count != len(userIDs)` の場合は `ErrNotFound` を返す。重複チェック（既にメンバー）は repository 層で行い、`ErrConflict` を返す。

`RemoveGroupMembers` は handler 層で `user_ids` の空チェック（`len == 0` → 400）を行い、service 層でまず `deduplicateUint64` によるユーザー ID 重複除去を行い、グループ存在確認後に repository へ委譲する。`deduplicateUint64` は `AddGroupMembers` / `RemoveGroupMembers` の両方で service 層に実装されており、COUNT 比較や `RowsAffected` 比較の正確性を保証する。

`CreateSubGroup` は service 層でバリデーション（自己参照・循環検出・コンポーネントサイズ上限・最大深度）を行い、`GroupRelationRepository.CreateRelation` を呼ぶ。`ErrConflict`（重複 INSERT）は `getStatusCode` 経由で 409 を返す。定数: `maxComponentGroups = 10`、`maxDepthNodes = 5`。

`DeleteSubGroup` は service 層のバリデーションなしで直接 `GroupRelationRepository.DeleteRelation` を呼ぶ。repository 層で `RowsAffected() == 0` の場合は `ErrNotFound` を返す。handler 層では `parentGroupID`（`:id`）と `childGroupID`（`:childId`）の両方を `parsePathID` でパースし、認証ユーザーの存在確認を行ってから service を呼ぶ。成功時は `204 No Content` を返す。

## リポジトリインターフェース（`GroupRepository`、`GroupRelationRepository`、`group.UserRepository`、`user.UserRepository`、`auth.UserRepository`）

それぞれのユースケース層が消費側でインターフェースを宣言する。

```go
// GroupRepository はグループデータアクセスのインターフェース（group/service.go で宣言）
type GroupRepository interface {
    ListGroups(ctx context.Context, q string, limit, offset int) ([]domain.Group, int, error)
    GetByID(ctx context.Context, id uint64) (domain.Group, error)
    ListGroupMembers(ctx context.Context, id uint64, limit, offset int, q string) ([]domain.GroupMember, int, error)
    Store(ctx context.Context, name, description string, userID uint64) (domain.Group, error)
    Update(ctx context.Context, id uint64, name, description string, userID uint64) (*domain.Group, error)
    Delete(ctx context.Context, id uint64, userID uint64) error
    ListNonGroupMembers(ctx context.Context, groupID uint64, limit, offset int, q string) ([]domain.User, int, error)
    AddGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) ([]domain.User, error)
    RemoveGroupMembers(ctx context.Context, groupID uint64, userIDs []uint64) error
}

// GroupRelationRepository はグループ間の親子関係データアクセスのインターフェース（group/service.go で宣言）
// MySQL 実装は internal/repository/mysql/group_relation.go の mysql.GroupRelationRepository
type GroupRelationRepository interface {
    GetAncestorIDs(ctx context.Context, groupID uint64) ([]uint64, error)
    GetDescendantIDs(ctx context.Context, groupID uint64) ([]uint64, error)
    CountComponentGroups(ctx context.Context, groupID uint64) (int, error)
    MaxDepthInComponent(ctx context.Context, parentGroupID, childGroupID uint64) (int, error)
    CreateRelation(ctx context.Context, parentGroupID, childGroupID uint64) (domain.GroupRelation, error)
    ListChildren(ctx context.Context, parentGroupID uint64) ([]domain.Group, error)
    DeleteRelation(ctx context.Context, parentGroupID, childGroupID uint64) error
}

// group.UserRepository はグループサービスが使うユーザーデータアクセスのインターフェース（group/service.go で宣言）
type UserRepository interface {
    CountByIDs(ctx context.Context, ids []uint64) (int, error)
}

// user.UserRepository はユーザー一覧取得・単件取得のインターフェース（user/service.go で宣言）
type UserRepository interface {
    ListUsers(ctx context.Context, q string, limit, offset int) ([]domain.User, int, error)
    GetByID(ctx context.Context, id uint64) (*domain.User, error)
}

// auth.UserRepository は認証サービスが使うユーザーデータアクセスのインターフェース（auth/service.go で宣言）
type UserRepository interface {
    GetByUUID(ctx context.Context, uuid string) (domain.User, error)
}
```

`Update` は DB の `groups` テーブルを `UPDATE groups SET name = ?, description = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL` で更新し、`RowsAffected() == 0` なら `ErrNotFound` を返す。更新後に `GetByID` で最新状態を取得して返す。`Delete` は `UPDATE groups SET deleted_at = NOW(), updated_by = ? WHERE id = ? AND deleted_at IS NULL` で soft delete し、`RowsAffected() == 0` なら `ErrNotFound` を返す。

`ListNonGroupMembers` は `users` テーブルから `group_members` に存在しないユーザーを返す。total は `q` フィルタ込みの非メンバー数（`q` が空の場合は全非メンバー数と一致する）。名前検索は `users.search_key` カラムへの LIKE 検索で行う。`search_key` は `CONCAT(first_name, last_name, last_name, first_name)` を値とする VIRTUAL GENERATED カラムで、`db/migrate/20260411120000_add_search_key_to_users.up.sql` で追加された。

`group.UserRepository.CountByIDs` は `mysql.UserRepository` が実装する。`CountByIDs` は `SELECT COUNT(DISTINCT id) FROM users WHERE id IN (?) AND deleted_at IS NULL` で存在するユーザー数を 1 クエリで返す。

`AddGroupMembers` はトランザクション内で `group_members` へ一括 INSERT する。INSERT 前に重複チェックを行い、既存メンバーが含まれる場合は `ErrConflict` を返す。成功後は追加したユーザーを `users` テーブルから SELECT して返す（`id, uuid, first_name, last_name` の全フィールドを SELECT する）。

`ListGroupMembers` は `domain.GroupMember` を返す。`GroupMember` は `id, uuid, first_name, last_name` に加え、`SourceGroups`（`[]domain.SourceGroup`）フィールドを持つ。`SourceGroup` は `GroupID` と `GroupName` を持ち、そのメンバーが所属する直属グループ（ルートの子グループ単位で集約）の情報を表す。MySQL の WITH RECURSIVE CTE で自グループと全子孫グループのメンバーを収集し、`JSON_ARRAYAGG` で各ユーザーの所属元グループをまとめて取得する。名前検索に使う `search_key` は repository 内部のローカル変数にスキャンし、`domain.GroupMember` には含まれない。`ListNonGroupMembers`・`AddGroupMembers` は `id, uuid, first_name, last_name` の全フィールドを SELECT する。`ListUsers` も同様に全フィールドを SELECT する。

`RemoveGroupMembers` は service 層でグループ存在確認を行い（`GetByID` 経由）、repository 層でトランザクション内に `DELETE FROM group_members WHERE group_id = ? AND user_id IN (?)` を実行する。`RowsAffected()` が `len(userIDs)` と一致しない場合（非メンバーが含まれる）は `ErrNotFound` を返してロールバックする。handler 層で `user_ids` の空チェック（`len == 0` → 400）を行う。成功時は `204 No Content` を返す。

> **補足**: `mysql.UserRepository` は `group.UserRepository`（`CountByIDs`）、`user.UserRepository`（`ListUsers`・`GetByID`）、`auth.UserRepository`（`GetByUUID`）の 3 つのインターフェースを実装する単一の struct。`GetByID` は `user.UserRepository` インターフェースの一部として `GET /api/v1/users/:id` のユーザー単件取得に使用される。`app/main.go` で `mysqlRepo.NewUserRepository(db)` で 1 インスタンスを生成し、`group.NewServiceWithRelation`・`user.NewService`・`auth.NewService` の 3 つに渡す。

`GroupRelationRepository` の各メソッドは WITH RECURSIVE CTE を使用する。`GetAncestorIDs` / `GetDescendantIDs` は有向グラフを再帰的に辿り祖先・子孫の ID を返す。`CountComponentGroups` は無向グラフとして辿り連結成分内のノード数を返す。`MaxDepthInComponent` は仮想エッジを追加した状態でルートから葉までの最大ノード数を返す。`ListChildren` は直属の子グループを `groups` テーブルから JOIN して返す。`CreateRelation` は UNIQUE 制約違反（エラーコード 1062）を `ErrConflict` にマッピングする。`DeleteRelation` は `DELETE FROM group_relations WHERE parent_group_id = ? AND child_group_id = ?` を実行し、`RowsAffected() == 0` の場合は `ErrNotFound` を返す。
