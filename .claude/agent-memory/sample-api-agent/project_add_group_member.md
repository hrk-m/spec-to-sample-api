---
name: add-group-member 機能の実装パターン
description: ListNonGroupMembers / AddGroupMembers の責務分担・重複チェック・トランザクションパターン
type: project
---

`GET /api/v1/groups/:id/non-members` と `POST /api/v1/groups/:id/members` の実装は 2026-04-11 時点で完了済み。

**責務分担:**
- service 層: グループ存在確認（GetByID）、ユーザー存在確認（GetUserByID）を担当
- repository 層: 重複メンバーチェック（ErrConflict）とトランザクション INSERT を担当

**Why:** 存在確認は service 層でビジネスロジックとして扱い、DB 制約違反（重複）は repository 層で吸収するパターン。

**How to apply:** 同様の「追加系」エンドポイントを実装する際は同じ責務分担に従う。service 層で参照整合チェック、repository 層で一意制約チェック。

**User 型の位置付け:**
- `domain.User`: non-members 一覧・add-group-members レスポンス・既存メンバー一覧（GET /api/v1/groups/:id/members）すべてで統一使用
- `domain.GroupMember` は 2026-04-11 に廃止。`domain/user.go` に `User` 型を独立定義し全箇所を統一
- `domain/group.go` には `Group` 型のみ残す

**non-members の検索:** `users.search_key` カラムへの LIKE 検索（`first_name LIKE ?` ではない）。
