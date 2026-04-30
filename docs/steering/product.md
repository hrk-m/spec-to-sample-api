# Product Steering

## 目的

`sample-api` は `spec-to-dev-workflow` リポジトリにおけるバックエンド API のリファレンス実装です。仕様（spec）から開発（dev）へのワークフローを検証するサンプルとして機能します。

## コア価値

- **参照実装**: Clean Architecture の実装パターンを具体的なコードで示す
- **ワークフロー検証**: 仕様書から実装への変換プロセスを実証する
- **拡張容易性**: 新しいドメイン機能を追加しやすい構造を維持する

## 主要機能

グループ管理機能を中心に、Clean Architecture パターンの実証を行います。

### ヘルスチェック

- `GET /health` — DB 接続を含むサーバーの稼働状態を返すエンドポイント
  - レスポンス: `{"status": "ok"}` (200) / `{"status": "error", "message": "db unavailable"}` (503)

### 認証

すべての `/api/v1/` エンドポイントは `AuthMiddleware` による認証が必要です。

- `GET /api/v1/me` — 認証済みユーザー自身の情報を返すエンドポイント
  - レスポンス: `{"id": uint64, "uuid": "string", "first_name": "string", "last_name": "string"}` (200)
  - エラー: 認証情報なし → 401
  - 認証方式: 開発環境（`APP_ENV=development`）では `DEV_USER_UUID` 環境変数で指定した UUID のユーザーを DB から取得してリクエストコンテキストにセット。本番環境向け認証（ALB OIDC 等）は未実装（実装時は `AuthMiddleware` 内で対応）

### グループ一覧取得

- `GET /api/v1/groups` — グループ一覧を返すエンドポイント
  - 任意パラメータ: `limit`（取得件数、1-500、デフォルト 500）、`offset`（オフセット、0 以上、デフォルト 0）、`q`（名前・説明の AND 検索、スペース区切り）
  - レスポンス: グループ一覧（groups）+ 総件数（total、`q` フィルタ込みの件数）

### グループ詳細取得

- `GET /api/v1/groups/:id` — 指定 ID のグループ詳細を返すエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - レスポンス: グループ情報（id, name, description, member_count, subgroups）。`subgroups` は直属の子グループ一覧（各要素: id, name, description, member_count）で、子グループが存在しない場合は空配列を返す
  - エラー: 不正な ID → 400、存在しない ID → 404

### グループ作成

- `POST /api/v1/groups` — 新しいグループを作成するエンドポイント
  - リクエストボディ: `{"name": "string", "description": "string"}`
  - バリデーション: name は必須かつ 100 文字以内（前後の空白はトリム）
  - 内部動作: トランザクション内で `groups` に INSERT し、認証ユーザーを `group_members` に自動追加する。`groups.updated_by` に認証ユーザーの ID をセットする
  - レスポンス: 作成されたグループ情報（id, name, description, member_count=1）(201)
  - エラー: リクエスト不正 → 400、バリデーション失敗（空文字・100 文字超過）→ 400

### グループ更新

- `PUT /api/v1/groups/:id` — 指定 ID のグループ名・説明を更新するエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - リクエストボディ: `{"name": "string", "description": "string"}`
  - バリデーション: name は必須かつ 100 文字以内（前後の空白はトリム）
  - レスポンス: 更新後のグループ情報（id, name, description, member_count）(200)
  - エラー: 不正な ID/パラメータ → 400、バリデーション失敗 → 400、存在しない ID → 404

### グループ削除

- `DELETE /api/v1/groups/:id` — 指定 ID のグループを soft delete するエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - レスポンス: 204 No Content
  - エラー: 不正な ID → 400、存在しない ID → 404

### グループメンバー一覧取得

- `GET /api/v1/groups/:id/members` — 指定グループ（自グループ＋全子孫グループ）のメンバー一覧を返すエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - 任意パラメータ: `limit`（取得件数、1-500、デフォルト 500）、`offset`（オフセット、0 以上、デフォルト 0）、`q`（名前検索）
  - レスポンス: メンバー一覧（members）+ 総件数（total）。各 member オブジェクトは `id, uuid, first_name, last_name` に加え `source_groups`（`[{group_id, group_name}]`）を含む。`source_groups` はそのユーザーが所属する直属の子グループ単位で集約した所属元グループ情報
  - エラー: 不正な ID/パラメータ → 400、グループ未存在 → 404

### グループ未所属ユーザー一覧取得

- `GET /api/v1/groups/:id/non-members` — 指定グループに所属していないユーザー一覧を返すエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - 任意パラメータ: `limit`（取得件数、1-500、デフォルト 500）、`offset`（オフセット、0 以上、デフォルト 0）、`q`（名前検索、`search_key` カラムで LIKE 検索）
  - レスポンス: ユーザー一覧（users）+ 総件数（total、`q` フィルタ込みの非メンバー数）
  - エラー: 不正な ID/パラメータ → 400、グループ未存在 → 404

### ユーザー一覧取得

- `GET /api/v1/users` — アクティブなユーザー一覧を返すエンドポイント
  - 任意パラメータ: `limit`（取得件数、1-500、デフォルト 500）、`offset`（オフセット、0 以上、デフォルト 0）、`q`（名前検索、`search_key` カラムで LIKE 検索）
  - レスポンス: ユーザー一覧（users）+ 総件数（total、`q` フィルタ込みの件数。`q` 未指定時は全アクティブユーザー数と等しい）
  - エラー: 不正なパラメータ → 400

### ユーザー詳細取得

- `GET /api/v1/users/:id` — 指定 ID のユーザー詳細を返すエンドポイント
  - パスパラメータ: `id`（ユーザー ID、1 以上の整数）
  - レスポンス: ユーザー情報（id, uuid, first_name, last_name）(200)
  - エラー: 不正な ID → 400、存在しない ID → 404

### グループメンバー追加

- `POST /api/v1/groups/:id/members` — 指定グループにユーザーを追加するエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - リクエストボディ: `{"user_ids": [uint64, ...]}`
  - バリデーション: `user_ids` は空でないこと。指定ユーザーが存在すること。指定ユーザーが既にグループメンバーでないこと
  - レスポンス: 追加されたユーザー一覧（members）(201)
  - エラー: 不正な ID/パラメータ → 400、グループ未存在 → 404、ユーザー未存在 → 404、既にメンバー → 409

### グループメンバー削除

- `DELETE /api/v1/groups/:id/members` — 指定グループからユーザーを一括削除するエンドポイント
  - パスパラメータ: `id`（グループ ID、1 以上の整数）
  - リクエストボディ: `{"user_ids": [uint64, ...]}`
  - バリデーション: `user_ids` は空でないこと。指定ユーザーが全員グループメンバーであること（1 件でも非メンバーがいれば全失敗）
  - レスポンス: 204 No Content
  - エラー: 不正な ID/パラメータ → 400、グループ未存在 → 404、非メンバーの user_id が含まれる → 404

### サブグループ追加

- `POST /api/v1/groups/:id/subgroups` — 指定グループを親とするサブグループ関係を作成するエンドポイント
  - パスパラメータ: `id`（親グループ ID、1 以上の整数）
  - リクエストボディ: `{"child_group_id": uint64}`
  - バリデーション: `child_group_id` は 1 以上の整数であること。親グループと子グループが同一でないこと。循環参照が発生しないこと。コンポーネント内グループ数が 10 以下であること。最大パス長（ノード数）が 5 以下であること
  - 内部動作: `group_relations` テーブルに `(parent_group_id, child_group_id)` を INSERT する。WITH RECURSIVE CTE で祖先・子孫・コンポーネントサイズ・最大深度を検証してから INSERT する
  - レスポンス: 作成された関係情報（parent_group_id, child_group_id）(201)
  - エラー: 不正な ID/パラメータ（循環・サイズ超過・深度超過・自己参照含む）→ 400、親グループ未存在 → 404、子グループ未存在 → 404、既に関係が存在 → 409

### サブグループ削除

- `DELETE /api/v1/groups/:id/subgroups/:childId` — 指定グループ間の親子関係を削除するエンドポイント
  - パスパラメータ: `id`（親グループ ID、1 以上の整数）、`childId`（子グループ ID、1 以上の整数）
  - バリデーション: `id`・`childId` ともに 1 以上の整数であること
  - 内部動作: `group_relations` テーブルから `(parent_group_id = id, child_group_id = childId)` のレコードを DELETE する
  - レスポンス: 204 No Content
  - エラー: 不正な ID → 400、指定した親子関係が存在しない → 404

## ドメインモデル

- **Group**: id, name, description, member_count
- **User**: id, uuid, first_name, last_name
- **GroupRelation**: parent_group_id, child_group_id

> **補足**: `domain.User`（`id, uuid, first_name, last_name`）は未所属ユーザー一覧（`GET /api/v1/groups/:id/non-members`）、グループメンバー追加レスポンス（`POST /api/v1/groups/:id/members`）、認証レスポンス（`GET /api/v1/me`）で使用する。グループメンバー一覧（`GET /api/v1/groups/:id/members`）では `domain.GroupMember` を使用する。`GroupMember` は `domain.User` のフィールドに加え `Sources`（`[]domain.GroupMemberSource`）を持ち、所属元グループ情報（`GroupID, GroupName`）を含む。HTTP レスポンスでは `source_groups` キーで JSON 出力される。`uuid` フィールドは `users` テーブルの `uuid` カラム（VARCHAR(36), UNIQUE）に対応し、`db/migrate/20260415120000_add_uuid_to_users.up.sql` で追加された。

> **補足**: `GroupRelation` は `domain/group_relation.go` に定義されており、`POST /api/v1/groups/:id/subgroups` のレスポンスとして使用される。`group_relations` テーブルは `db/migrate/20260425000000_create_group_relations.up.sql` で作成された（UNIQUE KEY: `(parent_group_id, child_group_id)`、外部キー: `groups(id) ON DELETE CASCADE`）。

## ユーザーとユースケース

このサンプル API は主に開発者向けです。新しいドメインを追加する際の実装パターンと設計判断の参照先として使用します。
