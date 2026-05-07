---
name: GroupRepository への logger 注入パターン
description: internal/repository/mysql/ 配下の全 repository が *slog.Logger を持つ package-wide convention
type: project
---

`internal/repository/mysql/` 配下の全 repository（`GroupRepository`, `UserRepository`, `GroupRelationRepository`）は `*slog.Logger` フィールドを保持し、コンストラクタで受け取る。

- `NewGroupRepository(db, logger)` — 既存
- `NewUserRepository(db, logger)` — 2026-05-02 追加（B-K4 対応）
- `NewGroupRelationRepository(db, logger)` — 2026-05-02 追加（B-K4 対応）

`app/main.go` で生成した `logger` インスタンスを三者すべてに渡す。

各メソッドが `domain.ErrInternalServerError` を返す手前で原エラーを `r.logger.ErrorContext(ctx, "...", "error", err)` でログする。

**Why:** GroupRepository は複雑な DB 処理のためログが必要だった。B-K4 で UserRepository / GroupRelationRepository も DB エラーを握り潰していた問題を解消し、package-wide の convention に統一した。

**How to apply:** 新たなリポジトリを `internal/repository/mysql/` に追加する際は必ず `*slog.Logger` をコンストラクタ引数に加え、`app/main.go` から `logger` を渡す。エラーログのメッセージ形式は `"{MethodName} query/scan/rows error failed"`, フィールドは `"error", err`。
