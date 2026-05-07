# Memory Index

- [add-group-member 機能の実装パターン](project_add_group_member.md) — ListNonGroupMembers / AddGroupMembers の責務分担・重複チェック・トランザクションパターン
- [repository logger 注入パターン](project_logger_injection.md) — internal/repository/mysql/ 全 repository が *slog.Logger を持つ package-wide convention（B-K4 で統一）
- [mocks/ type assertion スタイル](pattern_mock_type_assertion.md) — args.Get(N).(T) 直書き形式（fail-fast）に統一。comma-ok 形式は使わない
