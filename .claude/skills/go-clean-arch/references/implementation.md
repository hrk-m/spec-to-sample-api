# 実装パターン

> このファイルはテンプレート。最終的には `SKILL.md` の MUST ルールと repo-wide invariants を正とし、既存コードは観察元として使う。

## DB Migration / Seed の置き場

この skill では、schema 変更と seed データを分離して管理する。

### Migration（`db/migrate/`）

- ツール: `golang-migrate`
- ファイル命名: `YYYYMMDDHHMMSS_{table_name}.up.sql`
  - 例: `20260403120000_create_groups.up.sql`
- 内容: DDL のみ（CREATE TABLE, ALTER TABLE, DROP TABLE など）
- Makefile: `make db-migrate` / `make db-reset`（開発環境限定）

### Seed（`db/seed/`）

- ファイル命名: `seed.sql`（全テーブル分を FK 依存順にまとめる）
- 内容: DML のみ（INSERT など）、冪等に設計する
- Makefile: `make db-seed`

### 実行コードや runner の置き場

- `Makefile`, CI, Docker entrypoint
- 置かない場所: `domain/`, `{domain}/service.go`, `internal/rest/`, `internal/repository/mysql/`

既存 repo に `db/migrations/` の単発 SQL があっても、新規 schema 変更は `db/migrate/` へ、seed は `db/seed/` へ寄せる。

## 実装前チェック

新機能を追加する前に、まず次を行う。

1. 同じドメインの service / rest / repository / test を読む
2. 最も近い既存機能を 1 つ決める
3. その既存機能の naming、引数順、error 処理、テスト構成を真似る
4. 新しい抽象化を足す前に、既存ファイルへ素直に追加できないか確認する

このファイルのサンプルは出発点であって、実際の追加先では近傍コードが優先。

## 既存コード整列前チェック

既存コードをスキル準拠に寄せるときは、まず次を行う。

1. どの MUST ルールや invariant から外れているかを特定する
2. 変更単位を 1 file / 1 package / 1 usecase に閉じられるか確認する
3. 近傍コードの naming と責務分割は維持できるか確認する
4. mock と test を同じ変更で追随できるか確認する

整列の目的は「全置換」ではなく、「repo-wide ルールへ安全に寄せること」。

## 写経元の決め方

追加実装では、毎回「最も近い既存機能」を 1 つ決める。

優先順位:

1. 同じファイル内の近い責務
2. 同じ package 内の近い責務
3. 同じ層の別ドメイン実装
4. この reference のサンプル

合わせる対象:

- file 配置
- type 名、method 名
- 引数順
- error の返し方
- HTTP response 形式
- SQL の列順、改行、引数順
- test の粒度

合わせない対象:

- error 握り潰し
- close 漏れ、`rows.Err()` 未確認
- status map 漏れ
- テスト不足

写経元が `SKILL.md` の MUST ルールと矛盾する場合は、写経元ではなく整列対象として扱う。

## ディレクトリ構造

```text
db/
├── migrate/          ← schema 変更（DDL）, golang-migrate 形式
│   ├── 20260403120000_create_groups.up.sql
│   └── ...
└── seed/             ← 初期データ（DML のみ）
    └── seed.sql
{domain}/
├── mocks/
│   └── FooRepository.go   ← 手動作成・手動保守
├── service.go
└── service_test.go
internal/repository/mysql/
├── foo.go
└── foo_test.go
internal/rest/
├── foo.go
├── foo_test.go
└── mocks/
    └── FooService.go      ← 手動作成・手動保守
```

---

## 1. Entity（`domain/foo.go`）

```go
package domain

import "time"

type Foo struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name" validate:"required"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}
```

補足:

- 新規 timestamp は `time.Time` を優先
- 既存 entity の legacy な型は、移行タスクでない限り維持
- センチネルエラーは `domain/errors.go` に寄せる

---

## 2. Service + Repository IF（`foo/service.go`）

```go
package foo

import (
	"context"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
)

type FooRepository interface {
	Fetch(ctx context.Context, cursor string, num int64) ([]domain.Foo, string, error)
	GetByID(ctx context.Context, id int64) (domain.Foo, error)
	Store(ctx context.Context, f *domain.Foo) error
	Update(ctx context.Context, f *domain.Foo) error
	Delete(ctx context.Context, id int64) error
}

type Service struct {
	fooRepo FooRepository
}

func NewService(r FooRepository) *Service {
	return &Service{fooRepo: r}
}

func (s *Service) Fetch(ctx context.Context, cursor string, num int64) ([]domain.Foo, string, error) {
	return s.fooRepo.Fetch(ctx, cursor, num)
}

func (s *Service) GetByID(ctx context.Context, id int64) (domain.Foo, error) {
	return s.fooRepo.GetByID(ctx, id)
}

func (s *Service) Store(ctx context.Context, f *domain.Foo) error {
	return s.fooRepo.Store(ctx, f)
}

func (s *Service) Update(ctx context.Context, f *domain.Foo) error {
	f.UpdatedAt = time.Now()
	return s.fooRepo.Update(ctx, f)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	existing, err := s.fooRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == (domain.Foo{}) {
		return domain.ErrNotFound
	}
	return s.fooRepo.Delete(ctx, id)
}
```

補足:

- IF は消費側で宣言
- service 層は adapter 実装を import しない
- `time.Now()` 更新のような薄いユースケースロジックは service に置く
- repository mock は `mocks/` 配下に手動作成する
- 既存ドメインの機能追加なら、新しい service file を増やす前に既存 `service.go` への追加を優先する

---

## 3. Repository Adapter（`internal/repository/mysql/foo.go`）

```go
package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/repository"
)

type FooRepository struct {
	Conn *sql.DB
}

func NewFooRepository(conn *sql.DB) *FooRepository {
	return &FooRepository{Conn: conn}
}

func (m *FooRepository) fetch(ctx context.Context, query string, args ...interface{}) ([]domain.Foo, error) {
	rows, err := m.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	defer func() {
		if errRow := rows.Close(); errRow != nil {
			logrus.Error(errRow)
		}
	}()

	result := make([]domain.Foo, 0)
	for rows.Next() {
		item := domain.Foo{}
		err = rows.Scan(&item.ID, &item.Name, &item.UpdatedAt, &item.CreatedAt)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		logrus.Error(err)
		return nil, err
	}

	return result, nil
}

func (m *FooRepository) Fetch(ctx context.Context, cursor string, num int64) ([]domain.Foo, string, error) {
	query := `SELECT id, name, updated_at, created_at FROM foo WHERE created_at > ? ORDER BY created_at LIMIT ?`

	decodedCursor, err := repository.DecodeCursor(cursor)
	if err != nil && cursor != "" {
		return nil, "", domain.ErrBadParamInput
	}

	res, err := m.fetch(ctx, query, decodedCursor, num)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(res) == int(num) {
		nextCursor = repository.EncodeCursor(res[len(res)-1].CreatedAt)
	}
	return res, nextCursor, nil
}

func (m *FooRepository) Store(ctx context.Context, f *domain.Foo) error {
	query := `INSERT foo SET name=?, updated_at=?, created_at=?`
	stmt, err := m.Conn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, f.Name, f.UpdatedAt, f.CreatedAt)
	if err != nil {
		return err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	f.ID = lastID
	return nil
}

func (m *FooRepository) Update(ctx context.Context, f *domain.Foo) error {
	query := `UPDATE foo SET name=?, updated_at=? WHERE id = ?`
	stmt, err := m.Conn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, f.Name, f.UpdatedAt, f.ID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("weird behavior. total affected: %d", affected)
	}
	return nil
}
```

補足:

- mysql adapter は service / handler パッケージに依存しない
- cursor は helper に寄せる
- 近傍コードの query style は真似てよいが、resource の close や `rows.Err()` 確認のような安全性は落とさない
- 既存 repository 実装がこの safety line を満たしていなければ、legacy drift として整列対象にする
- レビュー時は「近傍コードとの一貫性」を先に確認する
- SQL の改行、列順、引数順も近傍クエリに合わせる

---

## 3-1. Bulk チェック・一括処理パターン

複数 ID の存在確認や重複チェックは、ループで単件 SELECT を呼ぶ（N+1）のではなく 1 クエリで処理する。

### 複数 ID の存在確認（`CountByIDs`）

```go
// ✅ 1 クエリで COUNT
func (r *UserRepository) CountByIDs(ctx context.Context, ids []uint64) (int, error) {
    if len(ids) == 0 {
        return 0, nil
    }
    placeholders := make([]string, len(ids))
    args := make([]interface{}, len(ids))
    for i, id := range ids {
        placeholders[i] = "?"
        args[i] = id
    }
    query := fmt.Sprintf( //nolint:gosec
        "SELECT COUNT(DISTINCT id) FROM users WHERE id IN (%s) AND deleted_at IS NULL",
        strings.Join(placeholders, ","),
    )
    var count int
    if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
        return 0, domain.ErrInternalServerError
    }
    return count, nil
}

// ❌ N+1（IDs の数だけ SELECT が走る）
for _, id := range ids {
    if _, err := r.GetByID(ctx, id); err != nil {
        return nil, err
    }
}
```

補足:

- `COUNT(DISTINCT id)` を使うことで、入力に重複 ID があっても誤カウントを防ぐ
- service 側で `count != len(deduplicatedIDs)` を確認して `ErrNotFound` を返す
- 重複 ID の除去（deduplication）は service 層で行い、repository には正規化済みのスライスを渡す

### 一括重複チェック（IN 句で 1 クエリ）

```go
// ✅ IN 句で一括チェック
placeholders := make([]string, len(userIDs))
checkArgs := make([]interface{}, 0, len(userIDs)+1)
checkArgs = append(checkArgs, groupID)
for i, uid := range userIDs {
    placeholders[i] = "?"
    checkArgs = append(checkArgs, uid)
}
checkQuery := fmt.Sprintf( //nolint:gosec
    "SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id IN (%s)",
    strings.Join(placeholders, ","),
)
var existingCount int
if err := r.db.QueryRowContext(ctx, checkQuery, checkArgs...).Scan(&existingCount); err != nil {
    return nil, domain.ErrInternalServerError
}
if existingCount > 0 {
    return nil, domain.ErrConflict
}

// ❌ ループで単件チェック（N+1）
for _, userID := range userIDs {
    var count int
    if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID).Scan(&count); err != nil {
        return nil, domain.ErrInternalServerError
    }
    if count > 0 {
        return nil, domain.ErrConflict
    }
}
```

### 一覧 API の COUNT クエリとフィルタの整合

一覧取得の `total` カウントは、SELECT と同じフィルタ条件を COUNT クエリにも適用する。

```go
// ✅ フィルタ込みの COUNT
countQuery := "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL"
var args []interface{}
if q != "" {
    countQuery += " AND search_key LIKE ?"
    args = append(args, "%"+q+"%")
}
var total int
r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

// ❌ フィルタなし COUNT（検索しても total が全件数になる）
countQuery := "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL"
r.db.QueryRowContext(ctx, countQuery).Scan(&total)
```

---

## 4. Handler + Service IF（`internal/rest/foo.go`）

```go
package rest

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	validator "gopkg.in/go-playground/validator.v9"

	"github.com/bxcodec/go-clean-arch/domain"
)

type FooService interface {
	Fetch(ctx context.Context, cursor string, num int64) ([]domain.Foo, string, error)
	GetByID(ctx context.Context, id int64) (domain.Foo, error)
	Store(ctx context.Context, f *domain.Foo) error
	Delete(ctx context.Context, id int64) error
}

type FooHandler struct {
	Service FooService
}

func NewFooHandler(e *echo.Echo, svc FooService) {
	h := &FooHandler{Service: svc}
	e.GET("/foos", h.Fetch)
	e.GET("/foos/:id", h.GetByID)
	e.POST("/foos", h.Store)
}

func isFooRequestValid(f *domain.Foo) (bool, error) {
	validate := validator.New()
	err := validate.Struct(f)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (h *FooHandler) GetByID(c echo.Context) error {
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
	}

	item, err := h.Service.GetByID(c.Request().Context(), int64(idP))
	if err != nil {
		return c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, item)
}
```

補足:

- Service IF は handler 側で宣言
- リクエスト入力エラーは handler が直接返す
- service 起点のエラーは `ResponseError` で返す
- 新しい domain エラーを client-visible にするなら `getStatusCode` も更新する
- service mock は `internal/rest/mocks/` 配下に手動作成する
- 既存 endpoint を増やすときは、route 登録順や handler メソッド名も近傍コードに揃える

---

## 5. DI（`app/main.go`）

```go
repo := mysqlRepo.NewFooRepository(dbConn)
svc := foo.NewService(repo)
rest.NewFooHandler(e, svc)
```

`app/main.go` では配線だけを行い、ユースケース判断は service に残す。

既存ドメインの機能追加で新しい依存が不要なら、`app/main.go` は触らない。
