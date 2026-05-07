---
name: mocks/ 配下の type assertion スタイル
description: mocks/ 配下では args.Get(N).(T) 直書き形式（fail-fast）に統一する
type: project
---

`group/mocks/` および `internal/rest/mocks/` 配下のモックメソッドでは、`args.Get(N).(T)` 直書き形式（fail-fast）を使う。comma-ok 形式（`val, _ := args.Get(N).(T)`）は使わない。

```go
// OK — fail-fast 直書き形式
return args.Get(0).([]domain.GroupMember), args.Int(1), args.Int(2), args.Error(3)

// NG — comma-ok 形式（silent failure になる）
members, _ := args.Get(0).([]domain.GroupMember)
return members, args.Int(1), args.Int(2), args.Error(3)
```

**Why:** 既存メソッドはすべて直書き形式（例: `args.Get(0).([]domain.Group)`）だった。panic は意図的なテスト失敗として扱う。`Return(nil, ...)` で untyped nil を渡すとパニックするが、typed nil（`[]domain.GroupMember(nil)`）では正常に動作する。

**How to apply:** 新規 mock メソッドを追加する際は必ず直書き形式で統一する。テストが `Return(nil, ...)` を使う場合は `Return([]T(nil), ...)` のように typed nil に変更する。
